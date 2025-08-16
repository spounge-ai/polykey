package persistence

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

type CircuitBreaker struct {
	maxFailures      int64
	resetTimeout     time.Duration
	halfOpenRequests int64

	state           atomic.Int32
	failures        atomic.Int64
	lastFailureTime atomic.Int64 // Unix nano
	successCount    atomic.Int64
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		maxFailures:      int64(maxFailures),
		resetTimeout:     resetTimeout,
		halfOpenRequests: int64(maxFailures) / 2,
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.canExecute() {
		// METRIC: Increment count of rejected requests
		return ErrCircuitBreakerOpen
	}

	err := fn()
	cb.recordResult(err)

	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	currentState := State(cb.state.Load())
	now := time.Now().UnixNano()

	switch currentState {
	case StateClosed:
		return true
	case StateOpen:
		lastFailure := cb.lastFailureTime.Load()
		if now > lastFailure+cb.resetTimeout.Nanoseconds() {
			// LOG: info, "circuit breaker state changing to half-open"
			// METRIC: Increment half-open event count
			if cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
				cb.successCount.Store(0)
			}
			return true // Allow the request
		}
		return false
	case StateHalfOpen:
		return cb.successCount.Load() < cb.halfOpenRequests
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(err error) {
	now := time.Now().UnixNano()
	currentState := State(cb.state.Load())

	if err != nil {
		// METRIC: Increment failure count
		cb.failures.Add(1)
		cb.lastFailureTime.Store(now)

		if currentState == StateHalfOpen || cb.failures.Load() >= cb.maxFailures {
			if cb.state.CompareAndSwap(int32(currentState), int32(StateOpen)) {
				// LOG: warn, "circuit breaker state changing to open"
				// METRIC: Increment open event count
			}
		}
	} else {
		// METRIC: Increment success count
		if currentState == StateHalfOpen {
			newSuccesses := cb.successCount.Add(1)
			if newSuccesses >= cb.halfOpenRequests {
				if cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateClosed)) {
					// LOG: info, "circuit breaker state changing to closed"
					// METRIC: Increment close event count
					cb.failures.Store(0)
				}
			}
		} else {
			cb.failures.Store(0)
		}
	}
}
