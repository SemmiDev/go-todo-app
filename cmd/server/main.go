package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/asynqtask"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/memcached"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/postgres"
	grpchandler "github.com/semmidev/go-todo-app/internal/adapter/driving/grpc"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/interceptor"
	httpdrv "github.com/semmidev/go-todo-app/internal/adapter/driving/http"
	authapp "github.com/semmidev/go-todo-app/internal/application/auth"
	todoapp "github.com/semmidev/go-todo-app/internal/application/todo"
	"github.com/semmidev/go-todo-app/internal/common/logging"
	"github.com/semmidev/go-todo-app/internal/common/ratelimit"
	"github.com/semmidev/go-todo-app/internal/common/token"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/common/validation"
	"github.com/semmidev/go-todo-app/internal/config"

	"buf.build/go/protovalidate"
)

func main() {
	cfg := config.Load()
	logger := logging.NewLogger(logging.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Environment,
		ServiceName: "todo-app",
	})
	slog.SetDefault(logger)

	if err := run(context.Background(), cfg, logger); err != nil {
		logger.Error("server exited with error", slog.Any("error", err))
		os.Exit(1)
	}
	logger.Info("server exited cleanly")
}

func run(ctx context.Context, cfg *config.AppConfig, logger *slog.Logger) error {
	logger.Info("config loaded", slog.Any("config", cfg.SafeLog()))

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer db.Close()
	logger.Info("postgres connected")

	// ─── Driven adapters (repositories) ──────────────────────────────────
	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	tagRepo := postgres.NewTagRepo(db)
	todoRepo := postgres.NewTodoRepo(db)
	todoTagRepo := postgres.NewTodoTagRepo(db)

	tokenMaker, err := token.NewPasetoMaker(cfg.TokenSymmetricKey)
	if err != nil {
		return fmt.Errorf("create token maker: %w", err)
	}

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisURL}
	taskDistributor := asynqtask.NewDistributor(redisOpt)

	uow := postgres.NewUnitOfWork(db)

	// ─── Application services (use-cases) ────────────────────────────────
	authSvc := authapp.NewService(userRepo, sessionRepo, tokenMaker, taskDistributor, authapp.Config{
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
		GoogleClientID:       cfg.GoogleClientID,
		GoogleClientSecret:   cfg.GoogleClientSecret,
		GoogleCallbackURL:    cfg.GoogleCallbackURL,
	}, db)
	memcachedRepo := memcached.NewCacheRepo(cfg.MemcachedURL)
	todoSvc := todoapp.NewService(todoRepo, tagRepo, todoTagRepo, memcachedRepo, uow)

	// ─── Validator ───────────────────────────────────────────────────────────
	val := validation.New()

	// ─── Start servers ───────────────────────────────────────────────────
	wg, ctx := errgroup.WithContext(ctx)
	runGRPCServer(ctx, wg, cfg, logger, authSvc, todoSvc, val)
	runGatewayServer(ctx, wg, cfg, logger)
	return wg.Wait()
}

func runGRPCServer(
	ctx context.Context,
	wg *errgroup.Group,
	cfg *config.AppConfig,
	logger *slog.Logger,
	authSvc *authapp.Service,
	todoSvc *todoapp.Service,
	val *validation.Validator,
) {
	limiter := ratelimit.NewRateLimiter(10, 20) // 10 rps, 20 burst

	protoValidator, err := protovalidate.New()
	if err != nil {
		logger.Error("failed to create protovalidate validator", slog.Any("error", err))
		return
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.RecoveryUnaryInterceptor(logger),
			interceptor.ValidatorUnaryInterceptor(protoValidator),
			interceptor.RateLimitUnaryInterceptor(limiter),
			interceptor.LoggingUnaryInterceptor(logger),
			interceptor.AuthUnaryInterceptor(authSvc),
		),
	)

	pb.RegisterAuthServiceServer(grpcSrv, grpchandler.NewAuthServer(authSvc, val))
	pb.RegisterTagServiceServer(grpcSrv, grpchandler.NewTagServer(todoSvc, val))
	pb.RegisterTodoServiceServer(grpcSrv, grpchandler.NewTodoServer(todoSvc, val))
	reflection.Register(grpcSrv)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Error("grpc listen failed", slog.Any("error", err))
		return
	}

	wg.Go(func() error {
		logger.Info("gRPC server listening", slog.String("port", cfg.GRPCPort))
		if err := grpcSrv.Serve(lis); err != nil {
			return fmt.Errorf("grpc serve: %w", err)
		}
		return nil
	})
	wg.Go(func() error {
		<-ctx.Done()
		grpcSrv.GracefulStop()
		return nil
	})
}

func runGatewayServer(
	ctx context.Context,
	wg *errgroup.Group,
	cfg *config.AppConfig,
	logger *slog.Logger,
) {
	limiter := ratelimit.NewRateLimiter(50, 100) // 50 rps, 100 burst for HTTP/Gateway

	handler, conn, err := httpdrv.NewRouter(ctx, httpdrv.RouterConfig{
		GRPCPort:    cfg.GRPCPort,
		Logger:      logger,
		RateLimiter: limiter,
	})
	if err != nil {
		logger.Error("setup http router failed", slog.Any("error", err))
		return
	}

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	wg.Go(func() error {
		logger.Info("HTTP gateway listening", slog.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http serve: %w", err)
		}
		return nil
	})
	wg.Go(func() error {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
		_ = conn.Close()
		return nil
	})
}
