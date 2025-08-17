package persistence

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
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
