package ratelimit

import (
	"sync"
	"golang.org/x/time/rate"
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
