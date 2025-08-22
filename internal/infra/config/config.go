package config

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-playground/validator/v10"
	infra_secrets "github.com/spounge-ai/polykey/internal/infra/secrets"
	"github.com/spounge-ai/polykey/internal/secrets"
	"github.com/spf13/viper"
	customvalidator "github.com/spounge-ai/polykey/pkg/validator"
	"gopkg.in/yaml.v3"
)

// BootstrapSecrets are loaded only from SSM Parameter Store
type BootstrapSecrets struct {
	PolykeyMasterKey string `secretpath:"polykey/kms/polykey_master_key"`
	NeonDBURL        string `secretpath:"polykey/db/neondb_url"`
	JWTRSAPrivateKey string `secretpath:"polykey/jwt/jwt_rsa_private_key"`
	TLSServerCert    string `secretpath:"polykey/tls/server-cert.pem"`
	TLSServerKey     string `secretpath:"polykey/tls/server-key.pem"`
	AWSKMSKeyARN     string `secretpath:"polykey/kms/aws_kms_key_arn"`
	SpoungeCA        string `secretpath:"tls/ca.pem"`

	// Dynamic config values
	CircuitBreakerConfig string `secretpath:"polykey/persistence/circuit_breaker"`
	RateLimiterConfig    string `secretpath:"polykey/server/rate_limiter"`
	AsyncAuditingConfig  string `secretpath:"polykey/auditing/asynchronous"`
}

// Config holds the runtime configuration
type Config struct {
	Server                   ServerConfig        `mapstructure:"server" validate:"required"`
	Persistence              PersistenceConfig   `mapstructure:"persistence" validate:"required"`
	AWS                      *AWSConfig          `mapstructure:"aws"`
	Authorization            AuthorizationConfig `mapstructure:"authorization" validate:"required"`
	ClientCredentialsPath    string              `mapstructure:"client_credentials_path"`
	DefaultKMSProvider       string              `mapstructure:"default_kms_provider" validate:"required,oneof=local aws vault"`
	BootstrapSecretsBasePath string              `mapstructure:"bootstrap_secrets_base_path" validate:"required"`
	Auditing                 AuditingConfig      `mapstructure:"auditing"`
	ServiceVersion   string
	BuildCommit      string
	BootstrapSecrets BootstrapSecrets
}

func Load(path string) (*Config, error) {
	vip := viper.New()
	setupViper(vip, path)

	if err := vip.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Load bootstrap secrets first if AWS is enabled
	var bootstrapSecrets *BootstrapSecrets
	var cfg Config
	if err := vip.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.AWS != nil && cfg.AWS.Enabled {
		var err error
		bootstrapSecrets, err = loadAWSBootstrapSecrets(&cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load bootstrap secrets: %w", err)
		}

		// Apply dynamic config overrides from bootstrap secrets
		if err := applyBootstrapConfigOverrides(vip, bootstrapSecrets); err != nil {
			return nil, fmt.Errorf("failed to apply bootstrap config overrides: %w", err)
		}
	}

	// Re-unmarshal config after applying overrides
	if err := vip.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config after bootstrap overrides: %w", err)
	}

	// Set bootstrap secrets
	if bootstrapSecrets != nil {
		cfg.BootstrapSecrets = *bootstrapSecrets
	}

	// Validate
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cfg.ServiceVersion = getenv("POLYKEY_SERVICE_VERSION", "unknown")
	cfg.BuildCommit = getenv("POLYKEY_BUILD_COMMIT", "unknown")

	return &cfg, nil
}

func setupViper(vip *viper.Viper, path string) {
	vip.SetEnvPrefix("POLYKEY")
	vip.AutomaticEnv()
	vip.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if path != "" {
		vip.SetConfigFile(path)
	} else {
		vip.SetConfigName("config.minimal")
		vip.AddConfigPath("./configs")
		vip.AddConfigPath(".")
	}
	vip.SetConfigType("yaml")

	setDefaults(vip)
}

func setDefaults(vip *viper.Viper) {
	vip.SetDefault("server.port", 50053)
	vip.SetDefault("server.mode", "development")
	vip.SetDefault("server.tls.enabled", true)
	vip.SetDefault("server.tls.client_auth", "RequireAndVerifyClientCert")

	vip.SetDefault("persistence.type", "neondb")

	vip.SetDefault("persistence.circuit_breaker.enabled", true)
	vip.SetDefault("persistence.circuit_breaker.max_failures", 5)
	vip.SetDefault("persistence.circuit_breaker.reset_timeout", "30s")

	vip.SetDefault("server.rate_limiter.enabled", true)
	vip.SetDefault("server.rate_limiter.rate", 10)
	vip.SetDefault("server.rate_limiter.burst", 20)

	vip.SetDefault("auditing.asynchronous.enabled", true)
	vip.SetDefault("auditing.asynchronous.channel_buffer_size", 10000)
	vip.SetDefault("auditing.asynchronous.worker_count", 3)
	vip.SetDefault("auditing.asynchronous.batch_size", 500)
	vip.SetDefault("auditing.asynchronous.batch_timeout", "1s")

	vip.SetDefault("aws.enabled", true)
	vip.SetDefault("aws.region", "us-east-1")

	vip.SetDefault("default_kms_provider", "local")
	vip.SetDefault("bootstrap_secrets_base_path", "/spounge/dev/")
	vip.SetDefault("authorization.zero_trust.enforce_mtls_identity_match", true)
}

func loadAWSBootstrapSecrets(cfg *Config) (*BootstrapSecrets, error) {
	awsCfg, err := aws_config.LoadDefaultConfig(
		context.Background(),
		aws_config.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	secretProvider := infra_secrets.NewParameterStore(awsCfg)
	return loadBootstrapSecrets(secretProvider, cfg.BootstrapSecretsBasePath)
}

// applyBootstrapConfigOverrides parses dynamic config from bootstrap secrets and applies to viper
func applyBootstrapConfigOverrides(vip *viper.Viper, secrets *BootstrapSecrets) error {
	// Apply circuit breaker config
	if secrets.CircuitBreakerConfig != "" {
		if err := applyConfigOverride(vip, "persistence.circuit_breaker", secrets.CircuitBreakerConfig); err != nil {
			return fmt.Errorf("failed to apply circuit breaker config: %w", err)
		}
	}

	// Apply rate limiter config
	if secrets.RateLimiterConfig != "" {
		if err := applyConfigOverride(vip, "server.rate_limiter", secrets.RateLimiterConfig); err != nil {
			return fmt.Errorf("failed to apply rate limiter config: %w", err)
		}
	}

	// Apply async auditing config
	if secrets.AsyncAuditingConfig != "" {
		if err := applyConfigOverride(vip, "auditing.asynchronous", secrets.AsyncAuditingConfig); err != nil {
			return fmt.Errorf("failed to apply async auditing config: %w", err)
		}
	}

	return nil
}

// applyConfigOverride parses a JSON/YAML config string and applies it to viper at the given key prefix
func applyConfigOverride(vip *viper.Viper, keyPrefix, configData string) error {
	configData = strings.TrimSpace(configData)
	if configData == "" {
		return nil
	}

	var config map[string]interface{}

	// Try JSON first, then YAML
	if err := json.Unmarshal([]byte(configData), &config); err != nil {
		if err := yaml.Unmarshal([]byte(configData), &config); err != nil {
			return fmt.Errorf("failed to parse config as JSON or YAML: %w", err)
		}
	}

	// Apply each key-value pair to viper
	for key, value := range config {
		fullKey := keyPrefix + "." + key
		vip.Set(fullKey, value)
	}

	return nil
}

func validateConfig(cfg *Config) error {
	validate := validator.New()
	if err := customvalidator.RegisterCustomValidators(validate); err != nil {
		return fmt.Errorf("failed to register custom validators: %w", err)
	}
	if err := validate.Struct(cfg); err != nil {
		return err
	}

	// Persistence check
	if cfg.Persistence.Type == "neondb" && cfg.BootstrapSecrets.NeonDBURL == "" {
		return fmt.Errorf("neondb URL required for neondb persistence (via bootstrap secrets)")
	}

	// Security checks
	if cfg.DefaultKMSProvider == "local" && cfg.BootstrapSecrets.PolykeyMasterKey == "" {
		return fmt.Errorf("polykey master key required for local KMS")
	}
	if cfg.BootstrapSecrets.JWTRSAPrivateKey == "" {
		return fmt.Errorf("JWT RSA private key is required")
	}
	if cfg.Server.TLS.Enabled {
		if cfg.BootstrapSecrets.TLSServerCert == "" {
			return fmt.Errorf("TLS cert required when TLS enabled")
		}
		if cfg.BootstrapSecrets.TLSServerKey == "" {
			return fmt.Errorf("TLS key required when TLS enabled")
		}
		if cfg.BootstrapSecrets.SpoungeCA == "" {
			return fmt.Errorf("CA cert required when TLS enabled")
		}

		// Validate TLS credentials
		if err := validateTLSCredentials(&cfg.BootstrapSecrets); err != nil {
			return fmt.Errorf("TLS credentials validation failed: %w", err)
		}
	}

	return nil
}

// validateTLSCredentials performs validation of TLS certificates and keys
func validateTLSCredentials(secrets *BootstrapSecrets) error {
	// Check for common PEM formatting issues
	if err := validatePEMFormat("TLS Server Cert", secrets.TLSServerCert, "CERTIFICATE"); err != nil {
		return err
	}
	
	if err := validatePEMFormat("TLS Server Key", secrets.TLSServerKey, "PRIVATE KEY"); err != nil {
		return err
	}
	
	if err := validatePEMFormat("CA Cert", secrets.SpoungeCA, "CERTIFICATE"); err != nil {
		return err
	}

	// Test actual TLS key pair loading
	_, err := tls.X509KeyPair([]byte(secrets.TLSServerCert), []byte(secrets.TLSServerKey))
	if err != nil {
		return fmt.Errorf("failed to load TLS key pair - cert/key mismatch or invalid format: %w", err)
	}

	return nil
}

// validatePEMFormat checks basic PEM structure
func validatePEMFormat(name, pemData, expectedType string) error {
	if pemData == "" {
		return fmt.Errorf("%s is empty", name)
	}

	// Trim whitespace
	trimmed := strings.TrimSpace(pemData)

	// Check for PEM header/footer
	expectedHeader := fmt.Sprintf("-----BEGIN %s-----", expectedType)
	expectedFooter := fmt.Sprintf("-----END %s-----", expectedType)
	
	if !strings.Contains(trimmed, expectedHeader) {
		// Check for alternative headers
		altHeaders := []string{
			"-----BEGIN RSA PRIVATE KEY-----",
			"-----BEGIN EC PRIVATE KEY-----",
			"-----BEGIN PRIVATE KEY-----",
		}
		
		found := false
		if expectedType == "PRIVATE KEY" {
			for _, header := range altHeaders {
				if strings.Contains(trimmed, header) {
					found = true
					break
				}
			}
		}
		
		if !found {
			return fmt.Errorf("%s missing expected PEM header %q (found headers: %v)", 
				name, expectedHeader, extractHeaders(trimmed))
		}
	}
	
	if !strings.Contains(trimmed, expectedFooter) && expectedType == "CERTIFICATE" {
		return fmt.Errorf("%s missing expected PEM footer %q", name, expectedFooter)
	}

	// Check for common encoding issues
	if strings.Contains(trimmed, "\\n") {
		return fmt.Errorf("%s contains escaped newlines - PEM data may be incorrectly encoded", name)
	}

	return nil
}

// extractHeaders finds PEM headers in the data for debugging
func extractHeaders(pemData string) []string {
	var headers []string
	lines := strings.Split(pemData, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-----BEGIN ") && strings.HasSuffix(line, "-----") {
			headers = append(headers, line)
		}
	}
	
	return headers
}

func getenv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func loadBootstrapSecrets(secretProvider secrets.BootstrapSecretProvider, basePath string) (*BootstrapSecrets, error) {
	secretsObj := &BootstrapSecrets{}
	secretsVal := reflect.ValueOf(secretsObj).Elem()
	secretsType := secretsVal.Type()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	base := strings.TrimRight(basePath, "/") + "/"

	for i := 0; i < secretsVal.NumField(); i++ {
		field := secretsVal.Field(i)
		fieldType := secretsType.Field(i)
		relPath := fieldType.Tag.Get("secretpath")

		if relPath == "" || !field.CanSet() {
			continue
		}

		fullPath := base + relPath
		secretValue, err := secretProvider.GetSecret(ctx, fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load secret %s (%s): %w", fieldType.Name, fullPath, err)
		}

		// Clean up common whitespace issues
		secretValue = strings.TrimSpace(secretValue)
		field.SetString(secretValue)
	}

	return secretsObj, nil
}