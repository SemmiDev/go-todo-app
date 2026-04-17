package interceptor

import (
	"context"

	"github.com/semmidev/go-todo-app/internal/common/ratelimit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimitUnaryInterceptor returns a new unary server interceptor that performs rate limiting.
func RateLimitUnaryInterceptor(rl *ratelimit.RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// In a real gRPC setup with a proxy, you'd extract the client IP from metadata (e.g., X-Forwarded-For).
		// For simplicity, we use a global limiter or a dummy "global" key if IP isn't easily accessible.
		// If you want per-IP gRPC limiting, extract it from peer.FromContext(ctx) or metadata.

		// For now, let's use a global limit for all gRPC calls to protect the server.
		// A more advanced version would use peer IP.
		if !rl.Allow("global") {
			return nil, status.Error(codes.ResourceExhausted, "too many requests")
		}

		return handler(ctx, req)
	}
}
