package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	customvalidator "github.com/spounge-ai/polykey/pkg/validator"
)

type Config struct {
	Server         ServerConfig   `mapstructure:"server"`
	Database       DatabaseConfig `mapstructure:"database" validate:"required"`
	Vault          VaultConfig    `mapstructure:"vault"    validate:"required"`
	AWS            AWSConfig      `mapstructure:"aws"      validate:"required"`
	ServiceVersion string
	BuildCommit    string
}

type AWSConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Region    string `mapstructure:"region"     validate:"required_if=Enabled true"`
	S3Bucket  string `mapstructure:"s3_bucket"  validate:"required_if=Enabled true"`
	KMSKeyARN string `mapstructure:"kms_key_arn" validate:"required_if=Enabled true,omitempty,arn"`
	CacheTTL  string `mapstructure:"cache_ttl"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port" validate:"required,gte=1024,lte=65535"`
	TLS  TLS    `mapstructure:"tls"`
	Mode string `mapstructure:"mode" validate:"required,oneof=development production"`
}

type TLS struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"     validate:"required"`
	Port     int    `mapstructure:"port"     validate:"required,gte=1024,lte=65535"`
	User     string `mapstructure:"user"     validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	DBName   string `mapstructure:"dbname"   validate:"required"`
	SSLMode  string `mapstructure:"sslmode"  validate:"required"`
}

type VaultConfig struct {
	Address string `mapstructure:"address" validate:"required,url"`
	Token   string `mapstructure:"token"   validate:"required"`
}

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

	vip.SetDefault("server.port", 50053)
	vip.SetDefault("aws.cache_ttl", "5m")

	if err := vip.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := vip.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	validate := validator.New()
	if err := customvalidator.RegisterCustomValidators(validate); err != nil {
		return nil, fmt.Errorf("failed to register custom validators: %w", err)
	}

	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cfg.ServiceVersion = getenv("POLYKEY_SERVICE_VERSION", "unknown")
	cfg.BuildCommit = getenv("POLYKEY_BUILD_COMMIT", "unknown")

	return &cfg, nil
}

// getenv returns an environment variable or a default value.
func getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}