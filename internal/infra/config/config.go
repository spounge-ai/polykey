package config

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-playground/validator/v10"
	infra_secrets "github.com/spounge-ai/polykey/internal/infra/secrets"
	"github.com/spounge-ai/polykey/internal/secrets"
	"github.com/spf13/viper"
	customvalidator "github.com/spounge-ai/polykey/pkg/validator"
)

type BootstrapSecrets struct {
	PolykeyMasterKey     string `secretpath:"/polykey/polykey_master_key"`
	NeonDBURLDevelopment string `secretpath:"/polykey/databases/neondb_url_development"`
	JWTRSAPrivateKey     string `secretpath:"/polykey/jwt_secret"`
}

type Config struct {
	Server           ServerConfig         `mapstructure:"server"`
	Persistence      PersistenceConfig    `mapstructure:"persistence"`
	NeonDB           *NeonDBConfig        `mapstructure:"neondb"`
	CockroachDB      *CockroachDBConfig   `mapstructure:"cockroachdb"`
	Vault            *VaultConfig         `mapstructure:"vault"`
	AWS              *AWSConfig           `mapstructure:"aws"`
	Auditing         AuditingConfig       `mapstructure:"auditing"`
	Authorization        AuthorizationConfig  `mapstructure:"authorization"`
	ClientCredentialsPath string               `mapstructure:"client_credentials_path"`
	DefaultKMSProvider    string               `mapstructure:"default_kms_provider"`
	StorageBackend       string               `mapstructure:"storage_backend"`
	ServiceVersion   string
	BuildCommit      string
	BootstrapSecrets BootstrapSecrets
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
	vip.SetDefault("storage_backend", "neondb")
	vip.SetDefault("default_kms_provider", "local")

	vip.SetDefault("persistence.circuit_breaker.enabled", true)
	vip.SetDefault("persistence.circuit_breaker.max_failures", 5)
	vip.SetDefault("persistence.circuit_breaker.reset_timeout", "30s")

	vip.SetDefault("auditing.asynchronous.enabled", true)
	vip.SetDefault("auditing.asynchronous.channel_buffer_size", 10000)
	vip.SetDefault("auditing.asynchronous.worker_count", 3)
	vip.SetDefault("auditing.asynchronous.batch_size", 500)
	vip.SetDefault("auditing.asynchronous.batch_timeout", "1s")

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

	cfg.StorageBackend = vip.GetString("STORAGE_BACKEND")

	if cfg.AWS.Enabled {
		awsCfg, err := aws_config.LoadDefaultConfig(context.Background(), aws_config.WithRegion(cfg.AWS.Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load aws config: %w", err)
		}
		secretProvider := infra_secrets.NewParameterStore(awsCfg)
		bootstrapSecrets, err := loadBootstrapSecrets(secretProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to load bootstrap secrets: %w", err)
		}
		cfg.BootstrapSecrets = *bootstrapSecrets
	}

	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if err := validatePersistence(&cfg); err != nil {
		return nil, err
	}

	if err := validateSecurity(&cfg); err != nil {
		return nil, err
	}

	cfg.ServiceVersion = getenv("POLYKEY_SERVICE_VERSION", "unknown")
	cfg.BuildCommit = getenv("POLYKEY_BUILD_COMMIT", "unknown")

	return &cfg, nil
}

func getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func loadBootstrapSecrets(secretProvider secrets.BootstrapSecretProvider) (*BootstrapSecrets, error) {
	secrets := &BootstrapSecrets{}
	secretsVal := reflect.ValueOf(secrets).Elem()
	secretsType := secretsVal.Type()

	for i := 0; i < secretsVal.NumField(); i++ {
		field := secretsVal.Field(i)
		fieldType := secretsType.Field(i)
		secretPath := fieldType.Tag.Get("secretpath")

		if secretPath == "" {
			continue
		}

		secretValue, err := secretProvider.GetSecret(context.Background(), secretPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load secret for %s: %w", fieldType.Name, err)
		}

		if field.CanSet() {
			field.SetString(secretValue)
		}
	}

	return secrets, nil
}

func validatePersistence(cfg *Config) error {
	switch cfg.Persistence.Type {
	case "cockroachdb":
		if cfg.CockroachDB == nil {
			return fmt.Errorf("persistence type is cockroachdb, but cockroachdb config is missing")
		}
	case "s3":
		if cfg.AWS == nil {
			return fmt.Errorf("persistence type is s3, but aws config is missing")
		}
	}
	return nil
}

func validateSecurity(cfg *Config) error {
	if cfg.DefaultKMSProvider == "local" && cfg.BootstrapSecrets.PolykeyMasterKey == "" {
		return fmt.Errorf("security validation failed: polykey master key is required for local KMS provider")
	}

	if cfg.BootstrapSecrets.JWTRSAPrivateKey == "" {
		return fmt.Errorf("security validation failed: JWT RSA private key is required")
	}

	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" {
			return fmt.Errorf("security validation failed: TLS cert file is required when TLS is enabled")
		}
		if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("security validation failed: TLS cert file not found at %s", cfg.Server.TLS.CertFile)
		}
		if cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("security validation failed: TLS key file is required when TLS is enabled")
		}
		if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("security validation failed: TLS key file not found at %s", cfg.Server.TLS.KeyFile)
		}
	}
	return nil
}