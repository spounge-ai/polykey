package config

// PersistenceConfig represents the persistence configuration.
type PersistenceConfig struct {
	Type string `mapstructure:"type" validate:"required,oneof=s3 neondb cockroachdb"`
}

// NeonDBConfig represents the NeonDB configuration.
type NeonDBConfig struct {
	URL string `mapstructure:"url" validate:"required,url"`
}

// CockroachDBConfig represents the CockroachDB configuration.
type CockroachDBConfig struct {
	URL string `mapstructure:"url" validate:"required,url"`
}
