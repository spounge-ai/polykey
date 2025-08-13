package persistence

import (
	"context"
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	maxFailures      int
	resetTimeout     time.Duration
	halfOpenRequests int

	mu              sync.Mutex
	state           State
	failures        int
	lastFailureTime time.Time
	successCount    int
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:      maxFailures,
		resetTimeout:     resetTimeout,
		halfOpenRequests: maxFailures / 2,
		state:            StateClosed,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.canExecute() {
		return ErrCircuitBreakerOpen
	}

	err := fn()
	cb.recordResult(err)

	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.After(cb.lastFailureTime.Add(cb.resetTimeout)) {
			cb.state = StateHalfOpen
			cb.successCount = 0
			return true
		}
		return false

	case StateHalfOpen:
		return cb.successCount < cb.halfOpenRequests

	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.state == StateHalfOpen || cb.failures >= cb.maxFailures {
			cb.state = StateOpen
		}
	} else {
		switch cb.state {
		case StateHalfOpen:
			cb.successCount++
			if cb.successCount >= cb.halfOpenRequests {
				cb.state = StateClosed
				cb.failures = 0
			}
		case StateClosed:
			cb.failures = 0
		}
	}
}
