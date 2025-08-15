package config

import "time"

// PersistenceConfig represents the persistence configuration.
type PersistenceConfig struct {
	Type     string         `mapstructure:"type" validate:"required,oneof=s3 neondb cockroachdb"`
	Database DatabaseConfig `mapstructure:"database"`
}

// DatabaseConfig represents the database configuration.
type DatabaseConfig struct {
	Connection DBConnectionConfig `mapstructure:"connection"`
	TLS        TLSConfig          `mapstructure:"tls"`
}

// DBConnectionConfig represents the database connection pool configuration.
type DBConnectionConfig struct {
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
}

// TLSConfig represents the database TLS configuration.
type TLSConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	SSLMode string `mapstructure:"ssl_mode"`
}

// NeonDBConfig represents the NeonDB configuration.
type NeonDBConfig struct {
	URL string `mapstructure:"url" validate:"required,url"`
}

// CockroachDBConfig represents the CockroachDB configuration.
type CockroachDBConfig struct {
	URL string `mapstructure:"url" validate:"required,url"`
}