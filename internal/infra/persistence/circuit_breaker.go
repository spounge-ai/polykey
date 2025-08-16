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

	switch currentState {
	case StateClosed:
		return true
	case StateOpen:
		now := time.Now().UnixNano()
		lastFailure := cb.lastFailureTime.Load()
		if now > lastFailure+cb.resetTimeout.Nanoseconds() {
			// Attempt to move to half-open state
			if cb.state.CompareAndSwap(int32(StateOpen), int32(StateHalfOpen)) {
				// TODO: Add logging and metrics for state change to half-open
				cb.successCount.Store(0)
			}
			// Allow the request that triggers the state change
			return true
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

	if err != nil {
		// METRIC: Increment failure count
		newFailures := cb.failures.Add(1)
		cb.lastFailureTime.Store(now)

		currentState := State(cb.state.Load())
		if currentState == StateHalfOpen || (currentState == StateClosed && newFailures >= cb.maxFailures) {
			// Trip the circuit breaker
			if cb.state.CompareAndSwap(int32(currentState), int32(StateOpen)) {
				// TODO: Add logging and metrics for state change to open
				_ = "noop"
			}
		}
	} else {
		// METRIC: Increment success count
		currentState := State(cb.state.Load())
		if currentState == StateHalfOpen {
			newSuccesses := cb.successCount.Add(1)
			if newSuccesses >= cb.halfOpenRequests {
				// Close the circuit breaker
				if cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateClosed)) {
					// TODO: Add logging and metrics for state change to closed
					cb.failures.Store(0)
				}
			}
		} else {
			// Reset failures on success in closed state
			cb.failures.Store(0)
		}
	}
}
