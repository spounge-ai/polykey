package google

import (
	"strconv")

type GeminiParams struct {
    TopP        float32
    Temperature float32
    TopK        int32
    MaxTokens   int32
}

func ParseGeminiParams(params map[string]string) GeminiParams {
	parseFloat32 := func(key string, def float32) float32 {
		if v, ok := params[key]; ok {
			if f64, err := strconv.ParseFloat(v, 32); err == nil {
				return float32(f64)
			}
		}
		return def
	}

	parseInt32 := func(key string, def int32) int32 {
		if v, ok := params[key]; ok {
			if i64, err := strconv.ParseInt(v, 10, 32); err == nil {
				return int32(i64)
			}
		}
		return def
	}

	return GeminiParams{
		TopP:        parseFloat32("top_p", 0.8),
		TopK:        parseInt32("top_k", 40),
		Temperature: parseFloat32("temperature", 0.7),
		MaxTokens:   parseInt32("max_tokens", 256),
	}
}
