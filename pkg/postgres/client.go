package postgres

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
)

// Client is a PostgreSQL client with connection pooling and prepared statements.
type Client struct {
	DB *pgxpool.Pool
}

// NewClient creates a new PostgreSQL client.
func NewClient(db *pgxpool.Pool) *Client {
	return &Client{DB: db}
}

// PrepareStatements prepares commonly used SQL statements for better performance.
func (c *Client) PrepareStatements(ctx context.Context, statements map[string]string) error {
	conn, err := c.DB.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for statement preparation: %w", err)
	}
	defer conn.Release()

	for name, sql := range statements {
		_, err := conn.Conn().Prepare(ctx, name, sql)
		if err != nil {
			return fmt.Errorf("failed to prepare statement %s: %w", name, err)
		}
	}

	return nil
}

// GetLockID generates a unique int64 lock ID from a KeyID for use with advisory locks.
func (c *Client) GetLockID(id domain.KeyID) int64 {
	h := fnv.New64a()
	h.Write([]byte(id.String()))
	return int64(h.Sum64())
}

// TryAcquireLock attempts to acquire a transaction-scoped advisory lock.
// It returns true if the lock was acquired, and false otherwise.
func (c *Client) TryAcquireLock(ctx context.Context, tx pgx.Tx, lockID int64) (bool, error) {
	var locked bool
	err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", lockID).Scan(&locked)
	if err != nil {
		return false, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}
	return locked, nil
}
