package config

// ServerConfig represents the server configuration.
type ServerConfig struct {
	Port int    `mapstructure:"port" validate:"required,gte=1024,lte=65535"`
	TLS  TLS    `mapstructure:"tls"`
	Mode string `mapstructure:"mode" validate:"required,oneof=development production"`
}

// TLS represents the TLS configuration.
type TLS struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}
