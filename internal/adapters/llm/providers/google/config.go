package google

/*
import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// Config holds the configuration for the Google LLM provider.
// Used by: NewGoogleProvider. Depends on: Application configuration loading.
type Config struct {
	// APIKey is required to initialize the Google AI client.
	APIKey string `json:"api_key" yaml:"api_key" validate:"required"`

	// Default settings for Gemini models - can be overridden by request parameters.
	DefaultModel       string  `json:"default_model" yaml:"default_model" validate:"required"`
	DefaultTemperature float64 `json:"default_temperature" yaml:"default_temperature"`
	DefaultMaxTokens   int     `json:"default_max_tokens" yaml:"default_max_tokens"`
	DefaultTimeoutSec  int     `json:"default_timeout_sec" yaml:"default_timeout_sec"`

	// Gemini-specific parameters
	DefaultTopP            float32 `json:"default_top_p" yaml:"default_top_p"`
	DefaultTopK            int     `json:"default_k" yaml:"default_k"`
	DefaultEmbeddingModel  string  `json:"default_embedding_model" yaml:"default_embedding_model"`
}

// NewDefaultConfig creates a Config with sensible defaults for Google Gemini.
// Used by: Application configuration loading. Depends on: None.
func NewDefaultConfig() Config {
	return Config{
		DefaultModel:          "gemini-2.5-flash-preview",
		DefaultTemperature:    0.7,
		DefaultMaxTokens:      500,
		DefaultTimeoutSec:     30,
		DefaultTopP:           1.0,
		DefaultTopK:           0, // 0 means disabled
		DefaultEmbeddingModel: "text-embedding-004",
	}
}

// Validate checks if the Google Config is valid using struct tags.
// Used by: NewGoogleProvider during initialization. Depends on: validator package.
func (c Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("google config validation failed: %w", err)
	}


	return nil
}


*/