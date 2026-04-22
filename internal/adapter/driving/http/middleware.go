// Package http provides the HTTP driving adapter, including route registration,
// template rendering, and middleware for the gateway server.
package http

import (
	"net/http"

	"github.com/semmidev/todo-app/internal/common/ratelimit"
)

// WithCORS wraps a handler with permissive CORS headers that reflect
// the request origin. Preflight OPTIONS requests are short-circuited.
func WithCORS(h http.Handler) http.Handler {
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

// WithRateLimit wraps a handler with a global token-bucket rate limiter.
// Requests that exceed the limit receive 429 Too Many Requests.
func WithRateLimit(h http.Handler, rl *ratelimit.RateLimiter) http.Handler {
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
