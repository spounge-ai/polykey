package persistence

import (
	"hash/fnv"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/pkg/postgres"
)

// PostgresBase provides a base implementation for PostgreSQL-backed repositories.
// It centralizes connection management, prepared statements, and other common logic.
type PostgresBase struct {
	*postgres.Client
	logger *slog.Logger
}

// NewPostgresBase creates a new PostgresBase.
func NewPostgresBase(db *pgxpool.Pool, logger *slog.Logger) *PostgresBase {
	return &PostgresBase{
		Client: postgres.NewClient(db),
		logger: logger,
	}
}

// GetLockID generates a unique int64 lock ID from a KeyID for use with advisory locks.
func (c *PostgresBase) GetLockID(id domain.KeyID) int64 {
	h := fnv.New64a()
	h.Write([]byte(id.String()))
	return int64(h.Sum64())
}