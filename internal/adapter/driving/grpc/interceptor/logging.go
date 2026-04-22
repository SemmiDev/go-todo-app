package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/semmidev/todo-app/internal/common/wideevent"
)

// LoggingUnaryInterceptor implements the Wide Event / Canonical Log Line pattern.
//
// Instead of emitting multiple narrow log lines throughout a request, it:
//  1. Initialises a WideEvent accumulator at the start of every request.
//  2. Pre-populates it with transport-level context (request_id, method, peer).
//  3. Passes the enriched context to downstream handlers — any layer can call
//     wideevent.Add(ctx, ...) to attach business fields without extra log lines.
//  4. After the handler returns, appends outcome fields (code, duration_ms, error).
//  5. Flushes the entire event as ONE structured log line (WARN for 4xx, ERROR
//     for 5xx, INFO for success).
//
// Reference: https://loggingsucks.com
func LoggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		// ── 1. Initialise wide event ──────────────────────────────────────────
		ctx = wideevent.New(ctx)

		// ── 2. Pre-populate with transport context ────────────────────────────
		reqID := generateRequestID()
		wideevent.Add(ctx,
			slog.String("request_id", reqID),
			slog.String("method", info.FullMethod),
			slog.String("peer_addr", peerAddr(ctx)),
			slog.String("user_agent", userAgent(ctx)),
		)

		// ── 3. Run the handler chain (auth, business logic, etc.) ─────────────
		resp, err := handler(ctx, req)

		// ── 4. Append outcome fields ──────────────────────────────────────────
		st, _ := status.FromError(err)
		durationMS := time.Since(start).Milliseconds()

		wideevent.Add(ctx,
			slog.String("code", st.Code().String()),
			slog.Int64("duration_ms", durationMS),
		)
		if err != nil {
			wideevent.Add(ctx, slog.String("error", st.Message()))
		}

		// ── 5. Choose log level based on outcome ─────────────────────────────
		level := outcomeLevel(st.Code())

		// ── 6. Flush — exactly ONE log line per request ───────────────────────
		wideevent.Flush(ctx, logger, "grpc request", level)

		return resp, err
	}
}

// RecoveryUnaryInterceptor catches panics in gRPC handlers and returns Internal error.
// It enriches the wide event with panic details before re-raising as gRPC Internal.
func RecoveryUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				wideevent.Add(ctx,
					slog.String("panic", fmt.Sprintf("%v", r)),
					slog.String("code", codes.Internal.String()),
				)
				wideevent.Flush(ctx, logger, "grpc panic", slog.LevelError)
				err = status.Error(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// outcomeLevel maps a gRPC code to the appropriate log level following the
// wide-event recommendation: always surface failures prominently.
func outcomeLevel(code codes.Code) slog.Level {
	switch code {
	case codes.OK:
		return slog.LevelInfo
	case codes.NotFound, codes.AlreadyExists, codes.InvalidArgument,
		codes.PermissionDenied, codes.Unauthenticated, codes.ResourceExhausted:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

// peerAddr extracts the remote address from the gRPC peer context.
func peerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return ""
}

// userAgent extracts the grpc-gateway forwarded user-agent (if present).
func userAgent(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	if ua := md.Get("grpcgateway-user-agent"); len(ua) > 0 {
		return ua[0]
	}
	if ua := md.Get("user-agent"); len(ua) > 0 {
		return ua[0]
	}
	return ""
}

// generateRequestID produces a short, URL-safe random ID (16 hex chars).
func generateRequestID() string {
	return fmt.Sprintf("req_%08x", rand.Int63())
}
