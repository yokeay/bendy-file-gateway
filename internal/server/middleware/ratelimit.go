package middleware

import (
	"sync"
	"time"

	"github.com/bendy/file-gateway/internal/types"
)

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// RateLimiter implements a token bucket algorithm.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64 // max tokens (burst)
}

// NewRateLimiter creates a rate limiter with the given rate and burst capacity.
func NewRateLimiter(ratePerSec, burst int) *RateLimiter {
	return &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     float64(ratePerSec),
		capacity: float64(burst),
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.capacity, lastFill: now}
		rl.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastFill).Seconds()
		b.tokens += elapsed * rl.rate
		if b.tokens > rl.capacity {
			b.tokens = rl.capacity
		}
		b.lastFill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RateLimit returns middleware that enforces per-IP and per-tenant rate limits.
// Default limits: 100 req/s per IP, 1000 req/s per tenant.
func RateLimit(perIP, perTenant int) types.Middleware {
	if perIP <= 0 {
		perIP = 100
	}
	if perTenant <= 0 {
		perTenant = 1000
	}

	ipLimiter := NewRateLimiter(perIP, perIP*2)
	tenantLimiter := NewRateLimiter(perTenant, perTenant*2)

	return func(next types.Handler) types.Handler {
		return func(req *types.Request) types.Response {
			ip := req.RemoteAddr
			if ip == "" {
				ip = "unknown"
			}

			if !ipLimiter.allow(ip) {
				return types.Error(429, "rate_limit_exceeded",
					"too many requests, please try again later", nil)
			}

			if req.TenantID != "" {
				if !tenantLimiter.allow(req.TenantID) {
					return types.Error(429, "rate_limit_exceeded",
						"tenant rate limit exceeded", nil)
				}
			}

			return next(req)
		}
	}
}
