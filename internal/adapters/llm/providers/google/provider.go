package google

import (
	"context"
  
	"fmt"
    "github.com/SpoungeAI/polykey-service/internal/adapters/llm"  // Import the generic LLM types
	"google.golang.org/genai"
)

// GoogleGeminiProvider implements the LLMProvider interface for Google's Gemini models
type GoogleGeminiProvider struct {
	client *genai.Client
	cfg *llm.ProviderConfig
}


func NewGoogleGeminiProvider(ctx context.Context, cfg *llm.ProviderConfig) (*GoogleGeminiProvider, error) {
    client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey: cfg.APIKey,
        Backend: genai.BackendGeminiAPI,
    })

    if err != nil {
        return nil, fmt.Errorf("failed to create Google Gemini client: %w", err)
    }

    return &GoogleGeminiProvider{
        client: client,
        cfg:    cfg,
    }, nil
}