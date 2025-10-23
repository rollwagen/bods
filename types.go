package main

import (
	"encoding/json"
	"slices"
	"strings"
)

type AnthropicModel int

const (
	UndefinedAnthropicModel AnthropicModel = iota
	ClaudeV3Sonnet
	ClaudeV3Haiku
	ClaudeV3Opus
	ClaudeV35Sonnet
	ClaudeV35SonnetV2
	ClaudeV35Haiku
	ClaudeV37Sonnet
	ClaudeV4Sonnet
	ClaudeV4Opus
	ClaudeV45Sonnet
	ClaudeV45Haiku
)

// Roles as defined by the Bedrock Anthropic Model API
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)

// The type of the content. Valid values are image and text.
const (
	MessageContentTypeText     = "text"
	MessageContentTypeImage    = "image"
	MessageContentTypeDocument = "document"
	MessageContentTypeToolUse  = "tool_use" // "type": "tool_use"
	MessageContentTypeThinking = "thinking"
)

// For content type 'image', the following image formats exist
const (
	MessageContentTypeMediaTypeJPEG = "image/jpeg"
	MessageContentTypeMediaTypePNG  = "image/png"
	MessageContentTypeMediaTypeWEBP = "image/webp"
	MessageContentTypeMediaTypeGIF  = "image/gif"
	MessageContentTypeMediaTypePDF  = "application/pdf"
)

// MessageContentTypes type of the image, possible image formats: jpeg, png, webp, gif
var MessageContentTypes = []string{
	MessageContentTypeMediaTypeJPEG,
	MessageContentTypeMediaTypePNG,
	MessageContentTypeMediaTypeWEBP,
	MessageContentTypeMediaTypeGIF,
	MessageContentTypeMediaTypePDF,
}

func (m AnthropicModel) IsClaude3OrHigherModel() bool {
	if m == ClaudeV3Sonnet || m == ClaudeV3Haiku || m == ClaudeV3Opus || m == ClaudeV35Sonnet || m == ClaudeV35SonnetV2 || m == ClaudeV37Sonnet || m == ClaudeV4Sonnet || m == ClaudeV4Opus || m == ClaudeV45Sonnet || m == ClaudeV45Haiku {
		return true
	}

	return false
}

func normalizeToModelID(id string) string {
	// remove region/global prefix if an inference profile id is given
	// handles both regional prefixes (eu., us.) and global prefix (global.)
	// examples: eu.anthropic.* -> anthropic.*, global.anthropic.* -> anthropic.*

	// Model IDs should start with "anthropic." - if they have a prefix before that, strip it
	if strings.Contains(id, ".anthropic.") {
		// Find the position of ".anthropic." and extract everything from "anthropic." onwards
		idx := strings.Index(id, ".anthropic.")
		modelID := id[idx+1:] // +1 to skip the leading dot
		logger.Printf("normalizeToModelID given id=%s, returning %s\n", id, modelID)
		return modelID
	}

	// No prefix found, return as-is
	logger.Printf("normalizeToModelID given id=%s, returning %s\n", id, id)
	return id
}

func IsClaude3OrHigherModelID(id string) bool {
	v3IDs := []string{
		ClaudeV3Sonnet.String(),
		ClaudeV3Haiku.String(),
		ClaudeV3Opus.String(),
		ClaudeV35Sonnet.String(),
		ClaudeV35SonnetV2.String(),
		ClaudeV35Haiku.String(),
		ClaudeV37Sonnet.String(),
		ClaudeV4Sonnet.String(),
		ClaudeV4Opus.String(),
		ClaudeV45Sonnet.String(),
		ClaudeV45Haiku.String(),
	}
	modelID := normalizeToModelID(id)
	return slices.Contains(v3IDs, modelID)
}

func IsVisionCapable(id string) bool {
	modelID := normalizeToModelID(id)
	return IsClaude3OrHigherModelID(modelID) && modelID != ClaudeV35Haiku.String()
}

// IsPromptCachingSupported returns true if the given model ID supports prompt caching.
// Prompt caching is generally available with Claude 3.7 Sonnet, Claude 3.5 Haiku, Claude 4, and Claude 4.5.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/prompt-caching.html#prompt-caching-models
func IsPromptCachingSupported(id string) bool {
	modelID := normalizeToModelID(id)
	cachingSupportedModels := []string{
		ClaudeV35Haiku.String(),  // Claude 3.5 Haiku
		ClaudeV37Sonnet.String(), // Claude 3.7 Sonnet
		ClaudeV4Sonnet.String(),  // Claude 4 Sonnet
		ClaudeV4Opus.String(),    // Claude 4 Opus
		ClaudeV45Sonnet.String(), // Claude 4.5 Sonnet
		ClaudeV45Haiku.String(),  // Claude 4.5 Haiku
	}
	return slices.Contains(cachingSupportedModels, modelID)
}

// IsClaude45Model returns true if the given model ID is Claude 4.5 (Sonnet or Haiku).
// Claude 4.5 models have a breaking change where only temperature OR top_p can be specified, not both.
func IsClaude45Model(id string) bool {
	modelID := normalizeToModelID(id)
	claude45Models := []string{
		ClaudeV45Sonnet.String(), // Claude 4.5 Sonnet
		ClaudeV45Haiku.String(),  // Claude 4.5 Haiku
	}
	return slices.Contains(claude45Models, modelID)
}

func (m AnthropicModel) String() string {
	switch m {
	case ClaudeV3Sonnet:
		return "anthropic.claude-3-sonnet-20240229-v1:0"
	case ClaudeV3Haiku:
		return "anthropic.claude-3-haiku-20240307-v1:0"
	case ClaudeV3Opus:
		return "anthropic.claude-3-opus-20240229-v1:0"
	case ClaudeV35Sonnet:
		return "anthropic.claude-3-5-sonnet-20240620-v1:0"
	case ClaudeV35SonnetV2:
		return "anthropic.claude-3-5-sonnet-20241022-v2:0"
	case ClaudeV35Haiku:
		return "anthropic.claude-3-5-haiku-20241022-v1:0"
	case ClaudeV37Sonnet:
		return "anthropic.claude-3-7-sonnet-20250219-v1:0"
	case ClaudeV4Sonnet:
		return "anthropic.claude-sonnet-4-20250514-v1:0"
	case ClaudeV4Opus:
		return "anthropic.claude-opus-4-20250514-v1:0"
	case ClaudeV45Sonnet:
		return "anthropic.claude-sonnet-4-5-20250929-v1:0"
	case ClaudeV45Haiku:
		return "anthropic.claude-haiku-4-5-20251001-v1:0"
	default:
		panic("AnthropicModel String()  - unhandled default case")
	}
}

var AnthrophicModelsIDs = []string{
	// ClaudeV2.String(),
	// ClaudeV21.String(),
	ClaudeV3Sonnet.String(),
	ClaudeV3Haiku.String(),
	ClaudeV3Opus.String(),
	ClaudeV35Sonnet.String(),
	ClaudeV35SonnetV2.String(),
	ClaudeV35Haiku.String(),
	ClaudeV37Sonnet.String(),
	ClaudeV4Sonnet.String(),
	ClaudeV4Opus.String(),
	ClaudeV45Sonnet.String(),
	ClaudeV45Haiku.String(),
}

// --- anthropic.claude ----------------------------
// see https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters.html

type AnthropicClaudeInferenceParameters struct { // "anthropic.claude-v2"
	Prompt        string   `json:"prompt"`
	Temperature   float64  `json:"temperature" validate:"required,gte=0,lte=1.0"`
	MaxTokens     int      `json:"max_tokens_to_sample" validate:"required,gte=200,lte=4096"` // max tokens to use in generated response
	TopP          float64  `json:"top_p" validate:"required,gte=0,lte=1.0"`
	TopK          int      `json:"top_k" validate:"required,gte=0,lte=500"`
	StopSequences []string `json:"stop_sequences"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}
type Source struct {
	Type      string `json:"type,omitempty"`       // "base64"
	MediaType string `json:"media_type,omitempty"` // e.g. "image/jpeg" or "application/pdf"
	Data      string `json:"data,omitempty"`       // encoded image in base64
}
type CacheControl struct {
	Type string `json:"type,omitempty"`
}
type Citations struct {
	Enabled bool `json:"enabled"`
}

type Content struct {
	Type      string `json:"type"`                  // 'image' or 'text' or 'document' (for pdf)
	Text      string `json:"text,omitempty"`        //  if Type='text'
	ID        string `json:"id,omitempty"`          // tool_use
	ToolUseID string `json:"tool_use_id,omitempty"` // tool_use
	Name      string `json:"name,omitempty"`        // tool_use
	Content   string `json:"content,omitempty"`     // tool_use
	// Input     string `json:"input,omitempty"`       // if Type='tool_use', json string
	Input        json.RawMessage `json:"input,omitempty"`     // if Type='tool_use'
	Source       *Source         `json:"source,omitempty"`    // if Type = 'image' or 'document'
	Thinking     string          `json:"thinking,omitempty"`  // if Type = 'thinking'
	Signature    string          `json:"signature,omitempty"` // if Type = 'thinking'
	CacheControl *CacheControl   `json:"cache_control,omitempty"`
	Citations    *Citations      `json:"citations,omitempty"`
}

type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // budget_tokens is 1024 tokens
}

type AnthropicClaudeMessagesInferenceParameters struct {
	AnthropicVersion string          `json:"anthropic_version"`
	Messages         []Message       `json:"messages"`
	System           string          `json:"system,omitempty"`
	Temperature      float64         `json:"temperature"`
	MaxTokens        int             `json:"max_tokens"`
	TopP             *float64        `json:"top_p,omitempty"` // pointer allows omitting for Claude 4.5 models
	TopK             int             `json:"top_k,omitempty"` // recommended for advanced use cases only; usually enough to just use temp
	StopSequences    []string        `json:"stop_sequences,omitempty"`
	Thinking         *ThinkingConfig `json:"thinking,omitempty"`
	Tools            []any           `json:"tools,omitempty"`          // Tools for Claude (e.g., text editor)
	AnthropicBeta    []string        `json:"anthropic_beta,omitempty"` // "anthropic_beta": ["computer-use-2024-10-22"] or ["token-efficient-tools-2025-02-19"]
}

type PerformanceConfig struct {
	Latency string `json:"latency"` // “latency” : “standard | optimized”
}

func NewThinkingConfig() *ThinkingConfig {
	return &ThinkingConfig{
		Type:         "enabled",
		BudgetTokens: defaultThinkingTokens, // 1024
	}
}

func NewAnthropicClaudeInferenceParameters() *AnthropicClaudeInferenceParameters {
	return &AnthropicClaudeInferenceParameters{
		Temperature:   1.0,
		TopP:          0.999,
		TopK:          250,
		MaxTokens:     defaultMaxTokens,
		StopSequences: []string{},
	}
}

func NewAnthropicClaudeMessagesInferenceParameters() *AnthropicClaudeMessagesInferenceParameters {
	topP := 0.999
	return &AnthropicClaudeMessagesInferenceParameters{
		AnthropicVersion: "bedrock-2023-05-31",
		Temperature:      1.0,
		TopP:             &topP, // pointer to allow omitting for Claude 4.5 models
		MaxTokens:        defaultMaxTokens,
		StopSequences:    []string{},
		Thinking:         nil, // will be set explicitly if needed
	}
}

type AnthropicClaudeStreamingChunk struct {
	Completion string `json:"completion"`
	Stop       string `json:"stop"`
	StopReason string `json:"stop_reason"`
}

type AnthropicClaudeMessagesResponse struct {
	// Type can be e.g.
	//   "message_start", "content_block_start", "content_block_delta",
	//   "content_block_stop", "message_delta", "message_stop"
	Type string `json:"type"`

	// type: ""message_start""
	Message *struct {
		Content      []any  `json:"content"`
		ID           string `json:"id"`
		Model        string `json:"model"`
		Role         string `json:"role"`
		StopReason   any    `json:"stop_reason"`
		StopSequence any    `json:"stop_sequence"`
		Type         string `json:"type"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`

	// type: "message_start""
	Usage *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`

	// type: "content_block"
	ContentBlock *struct {
		Text  string `json:"text"`
		Type  string `json:"type"`
		Index int    `json:"index,omitempty"`
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
	} `json:"content_block,omitempty"`

	// type: "content_block_delta"
	Delta *struct {
		StopReason   string `json:"stop_reason,omitempty"`
		StopSequence any    `json:"stop_sequence,omitempty"`
		Type         string `json:"type,omitempty"`
		Text         string `json:"text,omitempty"`
		Thinking     string `json:"thinking,omitempty"`
		PartialJSON  string `json:"partial_json,omitempty"`
		Signature    string `json:"signature,omitempty"`
	} `json:"delta,omitempty"`

	Index int `json:"index,omitempty"`

	// type: "message_stop"
	AmazonBedrockInvocationMetrics *struct {
		FirstByteLatency  int `json:"firstByteLatency"`
		InputTokenCount   int `json:"inputTokenCount"`
		InvocationLatency int `json:"invocationLatency"`
		OutputTokenCount  int `json:"outputTokenCount"`
	} `json:"amazon-bedrock-invocationMetrics,omitempty"`
}

type ResponseEventType int

const (
	UndefinedEventTyp      ResponseEventType = iota
	EventMessageStart                        // message_start
	EventContentBlockStart                   // content_block_start
	EventPing                                // ping
	EventContentBlockDelta                   // content_block_delta
	EventContentBlockStop                    // content_block_stop
	EventMessageDelta                        // message_delta
	EventMessageStop                         // message_stop
)

func (r ResponseEventType) String() string {
	switch r {
	case EventMessageStart:
		return "message_start"
	case EventContentBlockStart:
		return "content_block_start"
	case EventPing:
		return "ping"
	case EventContentBlockDelta:
		return "content_block_delta"
	case EventContentBlockStop:
		return "content_block_stop"
	case EventMessageDelta:
		return "message_delta"
	case EventMessageStop:
		return "message_stop"
	default:
		return "unknown ResponseEventType"
	}
}
