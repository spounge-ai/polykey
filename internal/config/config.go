package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Vault    VaultConfig    `mapstructure:"vault"`
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Port int  `mapstructure:"port"`
	TLS  TLS  `mapstructure:"tls"`
	Mode string `mapstructure:"mode"`
}

// TLS holds the TLS configuration.
type TLS struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// DatabaseConfig holds the database configuration.
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// VaultConfig holds the Vault configuration.
type VaultConfig struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
}

// Load loads the configuration from a file and environment variables.
func Load(path string) (*Config, error) {
	vip := viper.New()
	if path != "" {
		vip.SetConfigFile(path)
	} else {
		vip.SetConfigName("config")
		vip.AddConfigPath("./configs")
		vip.AddConfigPath(".")
	}

	vip.SetConfigType("yaml")
	vip.AutomaticEnv()
	vip.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	vip.SetDefault("server.port", 50052)

	if err := vip.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := vip.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with environment variables if they exist
	if port := vip.GetInt("POLYKEY_GRPC_PORT"); port != 0 {
		cfg.Server.Port = port
	}

	return &cfg, nil
}

// Getenv returns an environment variable or a default value.
func Getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}