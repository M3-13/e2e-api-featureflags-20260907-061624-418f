package main

import (
	"net/http"
	"sync"
	"time"
)

// withRateLimit wraps next with a token-bucket rate limiter. It is a global
// limiter: it tracks no per-IP state and keeps no IP logging. rps must be > 0;
// a non-positive value disables limiting.
func withRateLimit(next http.Handler, rps int) http.Handler {
	return withTokenBucket(next, newTokenBucket(rps))
}

// withTokenBucket wraps next with a shared token bucket so that every route
// drawing from the same limiter shares a single global budget. A nil limiter
// disables rate limiting.
func withTokenBucket(next http.Handler, limiter *tokenBucket) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow() {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newTokenBucket(rps int) *tokenBucket {
	if rps <= 0 {
		return nil
	}
	return &tokenBucket{
		rate:     float64(rps),
		capacity: float64(rps),
		tokens:   float64(rps),
		last:     time.Now(),
	}
}

type tokenBucket struct {
	mu       sync.Mutex
	rate     float64
	capacity float64
	tokens   float64
	last     time.Time
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
