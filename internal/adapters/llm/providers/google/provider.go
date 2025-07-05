package google

import (
	"context"
  
	"fmt"
    "github.com/SpoungeAI/polykey-service/internal/adapters/llm"  
	"google.golang.org/genai"
    //geminipb "github.com/SpoungeAI/polykey-service/proto/google/gemini"
)

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


