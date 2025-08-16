package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

// Limiter defines the interface for a rate limiter.
// This allows for different implementations (e.g., in-memory, distributed).
type Limiter interface {
	// Allow checks if a request is allowed for a given identifier (e.g., client ID).
	Allow(identifier string) bool
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter.
// It creates a new limiter for each identifier with the given rate and burst size.
func NewInMemoryRateLimiter(r rate.Limit, b int) Limiter {
	return &inMemoryRateLimiter{
		rate:      r,
		burst:     b,
		clients:   make(map[string]*rate.Limiter),
	}
}

type inMemoryRateLimiter struct {
	rate      rate.Limit
	burst     int
	clients   map[string]*rate.Limiter
	mu        sync.Mutex
}

func (l *inMemoryRateLimiter) Allow(identifier string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.clients[identifier]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.clients[identifier] = limiter
	}

	return limiter.Allow()
}
