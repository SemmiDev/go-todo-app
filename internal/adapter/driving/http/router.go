package http

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/semmidev/go-todo-app/docs"
	pb "github.com/semmidev/go-todo-app/gen/todo/v1"
	"github.com/semmidev/go-todo-app/internal/adapter/driving/http/httperr"
	"github.com/semmidev/go-todo-app/internal/common/ratelimit"
	"github.com/semmidev/go-todo-app/web"
)

// RouterConfig holds the dependencies needed by NewRouter.
type RouterConfig struct {
	// GRPCPort is the port of the gRPC server to proxy to.
	GRPCPort string
	// Logger is used for template-rendering errors.
	Logger *slog.Logger
	// RateLimiter is applied as HTTP middleware.
	RateLimiter *ratelimit.RateLimiter
}

// NewRouter builds the fully configured HTTP handler with all routes registered.
// The returned handler is ready to be used with an http.Server.
// The caller is responsible for closing the returned *grpc.ClientConn when done.
func NewRouter(ctx context.Context, cfg RouterConfig) (http.Handler, *grpc.ClientConn, error) {
	// ─── gRPC-Gateway mux ────────────────────────────────────────────────
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
		return nil, nil, fmt.Errorf("dial grpc for gateway: %w", err)
	}

	// ─── Register gateway services ───────────────────────────────────────
	if err := pb.RegisterAuthServiceHandler(ctx, gwMux, conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("register auth gateway: %w", err)
	}
	if err := pb.RegisterTagServiceHandler(ctx, gwMux, conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("register tag gateway: %w", err)
	}
	if err := pb.RegisterTodoServiceHandler(ctx, gwMux, conn); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("register todo gateway: %w", err)
	}

	// ─── Swagger / static FS ─────────────────────────────────────────────
	swaggerFS, err := fs.Sub(docs.SwaggerFS, "swagger")
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("swagger fs: %w", err)
	}

	// ─── Templates ───────────────────────────────────────────────────────
	tmpls, err := parseTemplates()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("parse templates: %w", err)
	}

	// ─── Route registration ──────────────────────────────────────────────
	mux := http.NewServeMux()

	// gRPC-Gateway API routes
	mux.Handle("/v1/", gwMux)

	// Static assets
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.FS(swaggerFS))))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS))))

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Page routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := tmpls.index.ExecuteTemplate(w, "layout", nil); err != nil {
			cfg.Logger.Error("execute template index", slog.Any("error", err))
		}
	})

	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpls.dashboard.ExecuteTemplate(w, "layout", nil); err != nil {
			cfg.Logger.Error("execute template dashboard", slog.Any("error", err))
		}
	})

	mux.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		if err := tmpls.callback.ExecuteTemplate(w, "layout", nil); err != nil {
			cfg.Logger.Error("execute template callback", slog.Any("error", err))
		}
	})

	// ─── Wrap with middleware ────────────────────────────────────────────
	handler := WithCORS(WithRateLimit(mux, cfg.RateLimiter))

	return handler, conn, nil
}
