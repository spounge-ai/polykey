package execution

import (
	"context"
	"time"
	"math/rand"
)

// RetryableFunc is a function that can be retried.
// It returns a result of type T and an error._
// The error should be nil if the function was successful._
type RetryableFunc[T any] func(ctx context.Context) (T, error)

// WithRetry executes a function with a retry mechanism.
// It uses exponential backoff with jitter to space out retries._
func WithRetry[T any](ctx context.Context, maxRetries int, initialBackoff time.Duration, maxBackoff time.Duration, fn RetryableFunc[T]) (T, error) {
	var result T
	var err error

	for i := 0; i < maxRetries; i++ {
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}

		backoff := initialBackoff * (1 << i) // Exponential backoff
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		jitter := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(backoff + jitter)
	}

	return result, err
}
