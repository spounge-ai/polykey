package execution

import (
	"context"
	"time"
)

// WithTimeout runs a function with a timeout and returns its result and error.
// The provided function `fn` is expected to honor the context cancellation.
func WithTimeout[T any](ctx context.Context, timeout time.Duration, fn func(ctx context.Context) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return fn(ctx)
}
