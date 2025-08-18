package persistence

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TransactionManager provides a generic way to execute functions within a database transaction.
type TransactionManager[T any] struct {
	logger *slog.Logger
}

// NewTransactionManager creates a new TransactionManager.
func NewTransactionManager[T any](logger *slog.Logger) *TransactionManager[T] {
	return &TransactionManager[T]{logger: logger}
}

// ExecuteInTransaction executes the given function within a serializable transaction,
// automatically handling retries with exponential backoff for serialization errors.
func (tm *TransactionManager[T]) ExecuteInTransaction(
	ctx context.Context,
	db *pgxpool.Pool,
	fn func(context.Context, pgx.Tx) (T, error),
) (T, error) {
	var result T
	var err error
	var zero T

	const maxRetries = 5
	const baseDelay = 10 * time.Millisecond
	const maxDelay = 250 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		tx, txErr := db.BeginTx(ctx, pgx.TxOptions{
			IsoLevel: pgx.Serializable,
		})
		if txErr != nil {
			return zero, fmt.Errorf("failed to begin transaction: %w", txErr)
		}

		result, err = fn(ctx, tx)
		if err == nil {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				var pgErr *pgconn.PgError
				if errors.As(commitErr, &pgErr) && pgErr.Code == "40001" {
					err = commitErr // Set err to trigger retry logic
					tm.logger.WarnContext(ctx, "serialization error on commit, retrying", "attempt", i+1, "max_attempts", maxRetries)
				} else {
					_ = tx.Rollback(ctx)
					return zero, fmt.Errorf("failed to commit transaction: %w", commitErr)
				}
			} else {
				return result, nil // Success
			}
		}

		// Always rollback on error
		_ = tx.Rollback(ctx)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "40001" {
			tm.logger.WarnContext(ctx, "serialization error detected, retrying", "attempt", i+1, "max_attempts", maxRetries)
			delay := baseDelay * time.Duration(1<<uint(i))

			if delay > maxDelay {
				delay = maxDelay
			}
			jitter := time.Duration(rand.Intn(int(delay / 10)))
			time.Sleep(delay + jitter)
			continue // Retry
		}

		// For any other error, do not retry.
		return zero, fmt.Errorf("transaction failed: %w", err)
	}

	return zero, fmt.Errorf("transaction failed after %d retries: %w", maxRetries, err)
}
