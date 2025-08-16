package circuitbreaker

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

var ErrOpen = errors.New("circuit breaker is open")

// Breaker is a generic, thread-safe circuit breaker.
type Breaker[T any] struct {
	maxFailures      int64
	resetTimeout     time.Duration
	halfOpenRequests int64

	state           atomic.Int32
	failures        atomic.Int64
	lastFailureTime atomic.Int64 // Unix nano
	successCount    atomic.Int64
}

// New creates a new generic Circuit Breaker.
func New[T any](maxFailures int, resetTimeout time.Duration) *Breaker[T] {
	cb := &Breaker[T]{
		maxFailures:      int64(maxFailures),
		resetTimeout:     resetTimeout,
		halfOpenRequests: 1, // Allow one successful request in half-open state to close the circuit
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// Execute wraps a function call with the circuit breaker logic.
func (cb *Breaker[T]) Execute(ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	if !cb.canExecute() {
		var zero T
		return zero, ErrOpen
	}

	result, err := fn(ctx)
	cb.recordResult(err)

	return result, err
}

func (cb *Breaker[T]) canExecute() bool {
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
				cb.successCount.Store(0)
			}
			return true
		}
		return false
	case StateHalfOpen:
		return cb.successCount.Load() < cb.halfOpenRequests
	default:
		return false
	}
}

func (cb *Breaker[T]) recordResult(err error) {
	now := time.Now().UnixNano()

	if err != nil {
		newFailures := cb.failures.Add(1)
		cb.lastFailureTime.Store(now)

		currentState := State(cb.state.Load())
		if currentState == StateHalfOpen || (currentState == StateClosed && newFailures >= cb.maxFailures) {
			cb.state.CompareAndSwap(int32(currentState), int32(StateOpen))
			// TODO: Add logging and metrics for state change to open
		}
	} else {
		currentState := State(cb.state.Load())
		if currentState == StateHalfOpen {
			newSuccesses := cb.successCount.Add(1)
			if newSuccesses >= cb.halfOpenRequests {
				if cb.state.CompareAndSwap(int32(StateHalfOpen), int32(StateClosed)) {
					cb.failures.Store(0)
				}
			}
		} else {
			cb.failures.Store(0)
		}
	}
}