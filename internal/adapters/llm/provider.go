package llm

import "context"

type LLMProvider interface {
    Generate(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}
