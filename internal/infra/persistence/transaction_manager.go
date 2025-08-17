package persistence

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
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

// ExecuteInTransaction executes the given function within a transaction.
// It automatically handles begin, commit, and rollback.
func (tm *TransactionManager[T]) ExecuteInTransaction(
	ctx context.Context,
	db *pgxpool.Pool,
	fn func(context.Context, pgx.Tx) (T, error),
) (T, error) {
	var zero T
	tx, err := db.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rErr := tx.Rollback(ctx); rErr != nil &&
			!errors.Is(rErr, context.Canceled) &&
			!errors.Is(rErr, pgx.ErrTxClosed) {
			tm.logger.Error("failed to rollback transaction", "error", rErr)
		}
	}()

	result, err := fn(ctx, tx)
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return result, nil
}
