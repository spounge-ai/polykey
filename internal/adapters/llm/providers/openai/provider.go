package openai


// holding off dev here, testing gemini first, free tier 


type OpenAIConfig struct {
    APIKey      string  `json:"api_key" yaml:"api_key" validate:"required"`
    BaseURL     string  `json:"base_url" yaml:"base_url"` // Optional, defaults to OpenAI API base URL
    Model       string  `json:"model" yaml:"model" validate:"required"`
    Temperature float64 `json:"temperature" yaml:"temperature"` // Default temperature for generation
    MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`   // Default max tokens for generation
    TimeoutSec  int     `json:"timeout_sec" yaml:"timeout_sec"` // Request timeout in seconds
}

