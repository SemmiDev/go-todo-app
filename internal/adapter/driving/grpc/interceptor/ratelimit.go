package interceptor

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiter manages rate limits per IP address (or other key).
type RateLimiter struct {
	ips sync.Map
	r   rate.Limit
	b   int
}

// NewRateLimiter creates a new rate limiter with the specified requests per second (r) and burst (b).
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		r: r,
		b: b,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	limiter, ok := rl.ips.Load(ip)
	if !ok {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.ips.Store(ip, limiter)
	}
	return limiter.(*rate.Limiter)
}

// Allow reports whether a request is allowed by the rate limiter for the given key.
func (rl *RateLimiter) Allow(key string) bool {
	return rl.getLimiter(key).Allow()
}

// RateLimitUnaryInterceptor returns a new unary server interceptor that performs rate limiting.
func RateLimitUnaryInterceptor(rl *RateLimiter) grpc.UnaryServerInterceptor {
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
