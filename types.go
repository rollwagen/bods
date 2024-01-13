package main

import "strings"

// See https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters.html

//
// --- anthropic.claude ----------------------------
//

type AnthropicClaudeInferenceParameters struct { // "anthropic.claude-v2"
	Prompt        string   `json:"prompt"`
	Temperature   float64  `json:"temperature" validate:"required,gte=0,lte=1.0"`
	StopSequences []string `json:"stop_sequences"`
	TopK          int      `json:"top_k" validate:"required,gte=0,lte=500"`
	TopP          float64  `json:"top_p" validate:"required"`
	MaxTokens     int      `json:"max_tokens_to_sample" validate:"required,gte=200,lte=8000"`
}

func NewAnthropicClaudeInferenceParameters() *AnthropicClaudeInferenceParameters {
	return &AnthropicClaudeInferenceParameters{
		Temperature:   0.5,
		TopP:          1.0,
		TopK:          250,
		MaxTokens:     200,
		StopSequences: []string{},
	}
}

type AnthropicClaudeResponseBody struct {
	Completion string `json:"completion"`
	Stop       string `json:"stop"`
	StopReason string `json:"stop_reason"`
}

func (r *AnthropicClaudeResponseBody) Text() string {
	return strings.TrimSpace(r.Completion)
}

type AnthropicClaudeStreamingChunk struct {
	Completion string `json:"completion"`
	Stop       string `json:"stop"`
	StopReason string `json:"stop_reason"`
}
