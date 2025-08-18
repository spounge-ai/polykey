package persistence

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spounge-ai/polykey/internal/infra/config"
)

// NewSecureConnectionPool creates a new database connection pool with enhanced security settings.
func NewSecureConnectionPool(ctx context.Context, dbConfig config.NeonDBConfig, serverConfig config.ServerConfig, persistenceConfig config.PersistenceConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}

	// Apply secure settings from a new SecureConnectionConfig struct if it were defined
	// For now, we will use the existing config structs to apply some settings.
	if serverConfig.Mode == "production" && !persistenceConfig.Database.TLS.Enabled {
		return nil, fmt.Errorf("database connection must use TLS in production mode")
	}

	// Start NeonDB Optimizations
	// Add -pooler to the host if not present. Assumes endpoint format is 'ep-project-123456.region.aws.neon.tech'.
	if !strings.Contains(poolConfig.ConnConfig.Host, "-pooler") {
		parts := strings.SplitN(poolConfig.ConnConfig.Host, ".", 2)
		if len(parts) == 2 {
			poolConfig.ConnConfig.Host = parts[0] + "-pooler." + parts[1]
		}
	}

	// Use simple protocol for PgBouncer compatibility.
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["prefer_simple_protocol"] = "true"
	// End NeonDB Optimizations

	if persistenceConfig.Database.TLS.Enabled {
		poolConfig.ConnConfig.TLSConfig = &tls.Config{
			ServerName:         poolConfig.ConnConfig.Host,
			InsecureSkipVerify: false, // Always false in production
			MinVersion:         tls.VersionTLS12,
		}
	}

	poolConfig.MaxConns = persistenceConfig.Database.Connection.MaxConns
	poolConfig.MinConns = persistenceConfig.Database.Connection.MinConns
	poolConfig.MaxConnIdleTime = persistenceConfig.Database.Connection.MaxConnIdleTime
	poolConfig.MaxConnLifetime = persistenceConfig.Database.Connection.MaxConnLifetime
	poolConfig.HealthCheckPeriod = persistenceConfig.Database.Connection.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
