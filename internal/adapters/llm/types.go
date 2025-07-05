package llm

type LLMRequest struct {
	Prompt     string   // Text prompt for generation
	Text       string   // For embedding requests
	Inputs     []string // Optional batch inputs
	History    []Message // For chat-style requests
	
	Model       string   // Optional override of default model
	Temperature float64  // Optional override of default temperature
	MaxTokens   int      // Optional override of default max tokens
	TopP        float64  // Optional override of top-p
	TopK        int      // Optional override of top-k
	
	Provider    string   // Which provider to use (empty = default)
}

// Message represents a single turn in a conversation.
type Message struct {
	Role    string // e.g., "user", "assistant", "system"
	Content string
}

// LLMResponse represents a result from an LLM.
type LLMResponse struct {
	Content   string      // Main output
	Embedding []float32   // For embedding requests
	Usage     Usage       // Token usage metadata
	Messages  []Message   // For chat completions (streaming or history-based)
}

// Usage captures token/resource usage info.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

 