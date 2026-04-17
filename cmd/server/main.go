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

	"html/template"

	"github.com/semmidev/go-todo-app/docs"
	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/asynqtask"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/memcached"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/postgres"
	grpchandler "github.com/semmidev/go-todo-app/internal/adapter/driving/grpc"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/httperr"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/grpc/interceptor"
	authapp "github.com/semmidev/go-todo-app/internal/application/auth"
	todoapp "github.com/semmidev/go-todo-app/internal/application/todo"
	"github.com/semmidev/go-todo-app/internal/common/logging"
	"github.com/semmidev/go-todo-app/internal/common/token"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/common/validation"
	"github.com/semmidev/go-todo-app/internal/config"
	"github.com/semmidev/go-todo-app/web"

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
	limiter := interceptor.NewRateLimiter(10, 20) // 10 rps, 20 burst

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
	limiter := interceptor.NewRateLimiter(50, 100) // 50 rps, 100 burst for HTTP/Gateway

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

	tmplBase, err := template.ParseFS(web.TemplateFS, "layout.html")
	if err != nil {
		logger.Error("parse base template", slog.Any("error", err))
		return
	}
	tmplIndex, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "index.html")
	if err != nil {
		logger.Error("parse index template", slog.Any("error", err))
		return
	}
	tmplDashboard, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "dashboard.html")
	if err != nil {
		logger.Error("parse dashboard template", slog.Any("error", err))
		return
	}
	tmplCallback, err := template.Must(tmplBase.Clone()).ParseFS(web.TemplateFS, "callback.html")
	if err != nil {
		logger.Error("parse callback template", slog.Any("error", err))
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/v1/", gwMux)
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := tmplIndex.ExecuteTemplate(w, "layout", nil); err != nil {
			logger.Error("execute template index", slog.Any("error", err))
		}
	})

	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if err := tmplDashboard.ExecuteTemplate(w, "layout", nil); err != nil {
			logger.Error("execute template dashboard", slog.Any("error", err))
		}
	})

	mux.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		if err := tmplCallback.ExecuteTemplate(w, "layout", nil); err != nil {
			logger.Error("execute template callback", slog.Any("error", err))
		}
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      withCORS(withRateLimit(mux, limiter)),
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

func withRateLimit(h http.Handler, rl *interceptor.RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use a simple global limiter for now, similar to gRPC.
		// In production, use client IP from r.RemoteAddr or X-Forwarded-For.
		if !rl.Allow("global") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"too many requests"}`))
			return
		}
		h.ServeHTTP(w, r)
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
