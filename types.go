package main

import "strings"

type AnthropicModel int

const (
	Undefined AnthropicModel = iota
	ClaudeV1
	ClaudeV2
	ClaudeV21
)

func (m AnthropicModel) String() string {
	switch m {
	case ClaudeV1:
		return "anthropic.claude-v1"
	case ClaudeV2:
		return "anthropic.claude-v2"
	case ClaudeV21:
		return "anthropic.claude-v2:1"
	}
	return "unknown"
}

var AnthophicModelsIDs = []string{
	ClaudeV1.String(),
	ClaudeV2.String(),
	ClaudeV21.String(),
}

// --- anthropic.claude ----------------------------
// see https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters.html

type AnthropicClaudeInferenceParameters struct { // "anthropic.claude-v2"
	Prompt        string   `json:"prompt"`
	Temperature   float64  `json:"temperature" validate:"required,gte=0,lte=1.0"`
	MaxTokens     int      `json:"max_tokens_to_sample" validate:"required,gte=200,lte=4096"` // max tokens to use in generated response
	TopP          float64  `json:"top_p" validate:"required"`
	TopK          int      `json:"top_k" validate:"required,gte=0,lte=500"`
	StopSequences []string `json:"stop_sequences"`
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
