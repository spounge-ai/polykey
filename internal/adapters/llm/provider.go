package llm

import (
	"context"
	"fmt"
)

// LLMProvider defines the interface that all LLM providers must implement.
// This allows for easy swapping between different providers (OpenAI, Google, Anthropic, etc.)
type LLMProvider interface {
	// GenerateText generates text from a prompt
	GenerateText(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	
	// GenerateTextStream generates text with streaming response
	GenerateTextStream(ctx context.Context, req *LLMRequest) (LLMStreamResponse, error)
	
	// GenerateEmbedding generates vector embeddings for text
	GenerateEmbedding(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	
	// ChatCompletion handles conversational responses
	ChatCompletion(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	
	// ChatCompletionStream handles conversational responses with streaming
	ChatCompletionStream(ctx context.Context, req *LLMRequest) (LLMStreamResponse, error)
	
	// GetName returns the provider's name for logging/debugging
	GetName() string
	
	// Close cleans up any resources
	Close() error
}

// LLMStreamResponse represents a streaming response from an LLM
type LLMStreamResponse interface {
	// Next returns the next chunk of the response
	Next() (*LLMResponse, error)
	
	// Close closes the stream
	Close() error
}

// ProviderConfig contains configuration for any LLM provider
type ProviderConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	
	// Provider-specific fields
	ProjectID string // For Google Vertex AI
	Location  string // For Google Vertex AI
	Timeout   int    // Request timeout in seconds
}

// LLMManager handles multiple providers and routing
type LLMManager struct {
	providers map[string]LLMProvider
	defaultProviderName   string
}

// NewLLMManager creates a new LLM manager
func NewLLMManager() *LLMManager {
	return &LLMManager{
		providers: make(map[string]LLMProvider),
	}
}

// RegisterProvider adds a new provider to the manager
func (m *LLMManager) RegisterProvider(name string, provider LLMProvider) {
	m.providers[name] = provider
}

// SetDefault sets the default provider
func (m *LLMManager) SetDefault(name string) error {
	if _, exists := m.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	m.defaultProviderName = name
	return nil
}

// GetProvider returns a specific provider by name
func (m *LLMManager) GetProvider(name string) (LLMProvider, error) {
	if name == "" {
		name = m.defaultProviderName
	}
	
	provider, exists := m.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	
	return provider, nil
}

// GenerateText uses the specified provider (or default) to generate text
func (m *LLMManager) GenerateText(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	provider, err := m.GetProvider(req.Provider)
	if err != nil {
		return nil, err
	}
	
	return provider.GenerateText(ctx, req)
}
