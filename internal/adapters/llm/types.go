package llm


// Request/Response structs
type LLMRequest struct {
	Provider    string                 `json:"provider"`
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Messages    []Message              `json:"messages,omitempty"`
	ModelParams map[string]interface{} `json:"model_params,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	Candidates           []TextCandidate        `json:"candidates"`
	PromptTokenUsage     int32                  `json:"prompt_token_usage"`
	CompletionTokenUsage int32                  `json:"completion_token_usage"`
	TotalTokenUsage      int32                  `json:"total_token_usage"`
	Error                string                 `json:"error,omitempty"`
	Provider             string                 `json:"provider,omitempty"`
	Model                string                 `json:"model,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	Embeddings           [][]float32            `json:"embeddings,omitempty"`
}

// ProviderConfig contains the minimal configuration needed for any provider.
type ProviderConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
	Timeout   int // Request timeout in seconds
}
