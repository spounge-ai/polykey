package config

// ServerConfig represents the server configuration.
type ServerConfig struct {
	Port       int               `mapstructure:"port" validate:"required,gte=1024,lte=65535"`
	TLS        TLS               `mapstructure:"tls"`
	Mode       string            `mapstructure:"mode" validate:"required,oneof=development production"`
	RateLimiter RateLimiterConfig `mapstructure:"rate_limiter"`
}

// RateLimiterConfig holds the configuration for the gRPC rate limiter.
type RateLimiterConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	Rate    float64 `mapstructure:"rate"`
	Burst   int     `mapstructure:"burst"`
}

// TLS represents the TLS configuration.
type TLS struct {
	Enabled      bool   `mapstructure:"enabled"`
	CertFile     string `mapstructure:"cert_file"`
	KeyFile      string `mapstructure:"key_file"`
	ClientCAFile string `mapstructure:"client_ca_file"`
	ClientAuth   string `mapstructure:"client_auth"`
}
