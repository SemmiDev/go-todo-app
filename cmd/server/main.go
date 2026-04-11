package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/semmidev/go-todo-app/docs"
	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	grpchandler "github.com/semmidev/go-todo-app/internal/adapter/driving/grpc"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/httperr"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/interceptor"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/postgres"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/memcached"
	authapp "github.com/semmidev/go-todo-app/internal/application/auth"
	todoapp "github.com/semmidev/go-todo-app/internal/application/todo"
	"github.com/semmidev/go-todo-app/internal/common/logging"
	"github.com/semmidev/go-todo-app/internal/common/token"
	"github.com/semmidev/go-todo-app/internal/common/validation"
	"github.com/semmidev/go-todo-app/internal/config"
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
	userRepo    := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewSessionRepo(db)
	tagRepo     := postgres.NewTagRepo(db)
	todoRepo    := postgres.NewTodoRepo(db)
	todoTagRepo := postgres.NewTodoTagRepo(db)

	tokenMaker, err := token.NewPasetoMaker(cfg.TokenSymmetricKey)
	if err != nil {
		return fmt.Errorf("create token maker: %w", err)
	}

	// ─── Application services (use-cases) ────────────────────────────────
	authSvc := authapp.NewService(userRepo, sessionRepo, tokenMaker, authapp.Config{
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
		GoogleClientID:       cfg.GoogleClientID,
		GoogleClientSecret:   cfg.GoogleClientSecret,
		GoogleCallbackURL:    cfg.GoogleCallbackURL,
	})
	memcachedRepo := memcached.NewCacheRepo(cfg.MemcachedURL)
	todoSvc := todoapp.NewService(todoRepo, tagRepo, todoTagRepo, memcachedRepo)

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
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptor.RecoveryUnaryInterceptor(logger),
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
	gwMux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{UseProtoNames: true},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
		// RFC 7807 Problem Details for all HTTP errors (via grpc-gateway translation)
		runtime.WithErrorHandler(httperr.GatewayErrorHandler),
	)

	conn, err := grpc.NewClient(
		"passthrough:///localhost:"+cfg.GRPCPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("dial grpc for gateway failed", slog.Any("error", err))
		return
	}

	if err := pb.RegisterAuthServiceHandler(ctx, gwMux, conn); err != nil {
		logger.Error("register auth gateway", slog.Any("error", err))
		return
	}
	if err := pb.RegisterTagServiceHandler(ctx, gwMux, conn); err != nil {
		logger.Error("register tag gateway", slog.Any("error", err))
		return
	}
	if err := pb.RegisterTodoServiceHandler(ctx, gwMux, conn); err != nil {
		logger.Error("register todo gateway", slog.Any("error", err))
		return
	}

	swaggerFS, err := fs.Sub(docs.SwaggerFS, "swagger")
	if err != nil {
		logger.Error("swagger fs", slog.Any("error", err))
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/", gwMux)
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      withCORS(mux),
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



func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
