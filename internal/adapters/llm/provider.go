package llm

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/structpb"
)

// LLMProvider defines the interface that all LLM providers must implement
type LLMProvider interface {
	GenerateText(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	GenerateTextStream(ctx context.Context, req *LLMRequest) (LLMStreamResponse, error)
	GenerateEmbedding(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	ChatCompletion(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
	ChatCompletionStream(ctx context.Context, req *LLMRequest) (LLMStreamResponse, error)
	GetName() string
	Close() error
}

// LLMStreamResponse represents a streaming response from an LLM
type LLMStreamResponse interface {
	Next() (*LLMResponse, error)
	Close() error
}

// LLMManager handles multiple providers and routing
type LLMManager struct {
	providers           map[string]LLMProvider
	defaultProviderName string
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

// for dev testing, we make this for dev testing, but can move to worker engine for langchaining.
func (r *LLMRequest) ToProtoRequest() (*GenerateTextRequest, error) {
	req := &GenerateTextRequest{
		Provider: r.Provider,
		Model:    r.Model,
		Prompt:   r.Prompt,
	}
	
	if r.ModelParams != nil {
		params, err := structpb.NewStruct(r.ModelParams)
		if err != nil {
			return nil, fmt.Errorf("failed to convert model params: %w", err)
		}
		req.ModelParams = params
	}
	
	return req, nil
}

func FromProtoRequest(req *GenerateTextRequest) (*LLMRequest, error) {
	llmReq := &LLMRequest{
		Provider: req.Provider,
		Model:    req.Model,
		Prompt:   req.Prompt,
	}
	
	return llmReq, nil
}

func (r *LLMResponse) ToProtoResponse() *GenerateTextResponse {
	candidates := make([]*TextCandidate, len(r.Candidates))
	for i := range r.Candidates {
		candidates[i] = &TextCandidate{
			Output:     r.Candidates[i].Output,
			TokenCount: r.Candidates[i].TokenCount,
		}
	}

	return &GenerateTextResponse{
		Candidates:           candidates,
		PromptTokenUsage:     r.PromptTokenUsage,
		CompletionTokenUsage: r.CompletionTokenUsage,
		TotalTokenUsage:      r.TotalTokenUsage,
	}
}

func FromProtoResponse(resp *GenerateTextResponse) *LLMResponse {
	candidates := make([]TextCandidate, len(resp.Candidates))
	for i, c := range resp.Candidates {
		candidates[i] = TextCandidate{
			Output:     c.Output,
			TokenCount: c.TokenCount,
		}
	}
	
	return &LLMResponse{
		Candidates:           candidates,
		PromptTokenUsage:     resp.PromptTokenUsage,
		CompletionTokenUsage: resp.CompletionTokenUsage,
		TotalTokenUsage:      resp.TotalTokenUsage,
	}
}

 