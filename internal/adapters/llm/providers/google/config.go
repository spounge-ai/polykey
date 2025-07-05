package google

import (
	"github.com/SpoungeAI/polykey-service/internal/adapters/llm"
	"google.golang.org/protobuf/types/known/structpb"
)

type Config struct {
	Temperature     float64                `json:"temperature"`
	MaxOutputTokens int                    `json:"maxOutputTokens"`
	TopP            float64                `json:"topP"`
	TopK            int                    `json:"topK"`
	CandidateCount  int                    `json:"candidateCount"`
	StopSequences   []string               `json:"stopSequences"`
	SafetySettings  map[string]interface{} `json:"safetySettings,omitempty"`
}

func NewConfig(params *structpb.Struct) *Config {
	e := llm.NewParamExtractor(params)
	return &Config{
		Temperature:     e.Float64("temperature", 0.9),
		MaxOutputTokens: e.Int("max_output_tokens", 2048),
		TopP:            e.Float64("top_p", 1.0),
		TopK:            e.Int("top_k", 32),
		CandidateCount:  e.Int("candidate_count", 1),
		StopSequences:   e.StringSlice("stop_sequences", []string{}),
		SafetySettings:  e.Map("safety_settings", nil),
	}
}

func (c *Config) ToAPIRequest(prompt string) map[string]any {
	req := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":     c.Temperature,
			"maxOutputTokens": c.MaxOutputTokens,
			"topP":            c.TopP,
			"topK":            c.TopK,
			"candidateCount":  c.CandidateCount,
			"stopSequences":   c.StopSequences,
		},
	}
	
	if c.SafetySettings != nil {
		req["safetySettings"] = c.SafetySettings
	}
	
	return req
}