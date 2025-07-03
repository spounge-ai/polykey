package llm

import "time"

// LLMRequest: Defines generic parameters for LLM tasks (generation, embedding, etc.).
// Used by: LLMProvider implementations to receive task specifications.
// Depends on: internals/core service logic that initiates LLM calls.
type LLMRequest struct {
    Prompt string // Main text input.

    // Optional overrides for provider/model defaults. Pointers handle unset state.
    Model       *string        // Specific model name (e.g., "gemini-1.5-flash-latest")
    Temperature *float64       // Controls randomness (0.0 to 1.0)
    MaxTokens   *int           // Maximum number of tokens in the response
    Timeout     *time.Duration // Request timeout duration
    TopP        *float32       // Nucleus sampling parameter
    TopK        *int           // Top-k sampling parameter
}

// LLMResponse: Represents generic output from an LLM task.
// Used by: Consumers of LLMProvider (e.g., internals/core service) to receive results.
// Depends on: LLMProvider implementations that return task results.
type LLMResponse struct {
    GeneratedText string // Primary text output.
    // Add fields for other outputs (e.g., Embeddings []float32).
}

// LLMMessage: Represents a single turn in a conversation.
// Used for: Structuring chat history in multi-turn interactions.
// Depends on: LLMProvider implementations that support chat APIs.
type LLMMessage struct {
    Role    string // Role of the message sender (e.g., "user", "assistant", "system")
    Content string // The message content
}