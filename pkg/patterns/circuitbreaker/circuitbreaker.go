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
var ErrTimeout = errors.New("circuit breaker operation timed out")

// StateChangeCallback is a function that gets called when the circuit breaker's state changes.
type StateChangeCallback func(from, to State)

// Breaker is a generic, thread-safe, context-aware circuit breaker.
type Breaker[T any] struct {
	// Configuration
	maxFailures      int64
	resetTimeout     time.Duration
	callTimeout      time.Duration
	halfOpenRequests int64
	onStateChange    StateChangeCallback

	// Internal state
	state           atomic.Int32
	failures        atomic.Int64
	lastFailureTime atomic.Int64 // Unix nano
	successCount    atomic.Int64
}

// Option configures a Breaker.
type Option[T any] func(*Breaker[T])

// WithResetTimeout sets the duration the breaker remains open before transitioning to half-open.
func WithResetTimeout[T any](d time.Duration) Option[T] {
	return func(b *Breaker[T]) {
		b.resetTimeout = d
	}
}

// WithCallTimeout sets the timeout for each individual call made through the breaker.
func WithCallTimeout[T any](d time.Duration) Option[T] {
	return func(b *Breaker[T]) {
		b.callTimeout = d
	}
}

// WithHalfOpenRequests sets the number of successful requests required in the half-open state to close the circuit.
func WithHalfOpenRequests[T any](n int64) Option[T] {
	return func(b *Breaker[T]) {
		if n > 0 {
			b.halfOpenRequests = n
		}
	}
}

// WithStateChangeCallback sets a callback function to be executed when the breaker's state changes.
func WithStateChangeCallback[T any](cb StateChangeCallback) Option[T] {
	return func(b *Breaker[T]) {
		b.onStateChange = cb
	}
}

// New creates a new generic Circuit Breaker.
func New[T any](maxFailures int, opts ...Option[T]) *Breaker[T] {
	b := &Breaker[T]{
		maxFailures:      int64(maxFailures),
		resetTimeout:     5 * time.Second, // Default reset timeout
		callTimeout:      2 * time.Second, // Default call timeout
		halfOpenRequests: 1,
		onStateChange:    func(from, to State) {}, // No-op callback by default
	}

	for _, opt := range opts {
		opt(b)
	}

	b.state.Store(int32(StateClosed))
	return b
}

// Execute wraps a function call with the circuit breaker logic.
func (b *Breaker[T]) Execute(ctx context.Context, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	if !b.canExecute() {
		return zero, ErrOpen
	}

	// If the parent context has a deadline shorter than our call timeout, respect it.
	callCtx, cancel := context.WithTimeout(ctx, b.callTimeout)
	defer cancel()

	resultChan := make(chan T, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := fn(callCtx)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		b.recordResult(nil)
		return result, nil
	case err := <-errChan:
		b.recordResult(err)
		return zero, err
	case <-callCtx.Done():
		// Check if the cancellation came from our timeout or the parent context.
		err := callCtx.Err()
		if errors.Is(err, context.DeadlineExceeded) {
			b.recordResult(ErrTimeout)
			return zero, ErrTimeout
		}
		// If it was the parent context, just record a generic failure.
		b.recordResult(err)
		return zero, err
	}
}

func (b *Breaker[T]) canExecute() bool {
	currentState := State(b.state.Load())

	switch currentState {
	case StateClosed:
		return true
	case StateOpen:
		// Check if the reset timeout has passed.
		if time.Now().UnixNano() > b.lastFailureTime.Load()+b.resetTimeout.Nanoseconds() {
			b.transition(StateOpen, StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow a limited number of requests through.
		return b.successCount.Load() < b.halfOpenRequests
	default:
		return false
	}
}

func (b *Breaker[T]) recordResult(err error) {
	if err != nil {
		// Failure path
		newFailures := b.failures.Add(1)
		b.lastFailureTime.Store(time.Now().UnixNano())

		currentState := State(b.state.Load())
		if currentState == StateHalfOpen || (currentState == StateClosed && newFailures >= b.maxFailures) {
			b.transition(currentState, StateOpen)
		}
	} else {
		// Success path
		currentState := State(b.state.Load())
		if currentState == StateHalfOpen {
			if b.successCount.Add(1) >= b.halfOpenRequests {
				b.transition(StateHalfOpen, StateClosed)
			}
		} else {
			// Reset failures on any success in the closed state.
			b.failures.Store(0)
		}
	}
}

func (b *Breaker[T]) transition(from, to State) {
	if b.state.CompareAndSwap(int32(from), int32(to)) {
		// Reset counters on state change.
		switch to {
		case StateOpen:
			b.successCount.Store(0)
		case StateHalfOpen:
			b.successCount.Store(0)
		case StateClosed:
			b.failures.Store(0)
			b.successCount.Store(0)
		}

		// Fire the callback.
		b.onStateChange(from, to)
	}
}
