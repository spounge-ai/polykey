package config

import "time"

// AuditingConfig holds the configuration for auditing.
type AuditingConfig struct {
	Asynchronous AsynchronousAuditingConfig `mapstructure:"asynchronous"`
}

// AsynchronousAuditingConfig holds the configuration for the asynchronous logger.
type AsynchronousAuditingConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	ChannelBufferSize int           `mapstructure:"channel_buffer_size"`
	WorkerCount       int           `mapstructure:"worker_count"`
	BatchSize         int           `mapstructure:"batch_size"`
	BatchTimeout      time.Duration `mapstructure:"batch_timeout"`
}
