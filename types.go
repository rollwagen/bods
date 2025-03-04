package main

import (
	"slices"
	"strings"
)

type AnthropicModel int

const (
	UndefinedAnthropicModel AnthropicModel = iota
	// ClaudeV2
	// ClaudeV21
	ClaudeV3Sonnet
	ClaudeV3Haiku
	ClaudeV3Opus
	ClaudeV35Sonnet
	ClaudeV35SonnetV2
	ClaudeV35Haiku
	ClaudeV37Sonnet
)

// Roles as defined by the Bedrock Anthropic Model API
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)

// The type of the content. Valid values are image and text.
const (
	MessageContentTypeText  = "text"
	MessageContentTypeImage = "image"
)

// For content type 'image', the following image formats exist
const (
	MessageContentTypeMediaTypeJPEG = "image/jpeg"
	MessageContentTypeMediaTypePNG  = "image/png"
	MessageContentTypeMediaTypeWEBP = "image/webp"
	MessageContentTypeMediaTypeGIF  = "image/gif"
)

// MessageContentTypes type of the image, possible image formats: jpeg, png, webp, gif
var MessageContentTypes = []string{
	MessageContentTypeMediaTypeJPEG,
	MessageContentTypeMediaTypePNG,
	MessageContentTypeMediaTypeWEBP,
	MessageContentTypeMediaTypeGIF,
}

func (m AnthropicModel) IsClaude3Model() bool {
	if m == ClaudeV3Sonnet || m == ClaudeV3Haiku || m == ClaudeV3Opus || m == ClaudeV35Sonnet || m == ClaudeV35SonnetV2 || m == ClaudeV37Sonnet {
		return true
	}

	return false
}

func normalizeToModelID(id string) string {
	// remove region prefix if as id an inference profile id is given
	startsWithTwoLettersAndDot := func(s string) bool {
		if len(s) < 3 {
			return false
		}
		return strings.HasPrefix(s, s[0:2]+".")
	}

	modelID := id
	if startsWithTwoLettersAndDot(id) {
		modelID = id[3:]
	}

	logger.Printf("normalizeToModelID given id=%s, returning %s\n", id, modelID)

	return modelID
}

func IsClaude3ModelID(id string) bool {
	v3IDs := []string{
		ClaudeV3Sonnet.String(),
		ClaudeV3Haiku.String(),
		ClaudeV3Opus.String(),
		ClaudeV35Sonnet.String(),
		ClaudeV35SonnetV2.String(),
		ClaudeV35Haiku.String(),
		ClaudeV37Sonnet.String(),
	}
	modelID := normalizeToModelID(id)
	return slices.Contains(v3IDs, modelID)
}

func IsVisionCapable(id string) bool {
	modelID := normalizeToModelID(id)
	return IsClaude3ModelID(modelID) && modelID != ClaudeV35Haiku.String()
}

func (m AnthropicModel) String() string {
	switch m {
	// case ClaudeV2:
	// 	return "anthropic.claude-v2"
	// case ClaudeV21:
	// 	return "anthropic.claude-v2:1"
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

//	type Message struct {
//		Content string `json:"content"`
//		Role    string `json:"role"` // "user" or "assistnat"
//	}
//
//	type Message struct {
//		Content []struct {
//			Text string `json:"text"`
//			Type string `json:"type"`
//		} `json:"content"`
//		Role    string `json:"role"` // "user" or "assistnat"
//	}
type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}
type Source struct {
	Type      string `json:"type,omitempty"`       // "base64"
	MediaType string `json:"media_type,omitempty"` // e.g. "image/jpeg"
	Data      string `json:"data,omitempty"`       // encoded image in base64
}
type Content struct {
	Type   string  `json:"type"`             // 'image' or 'text'
	Text   string  `json:"text,omitempty"`   //  if Type='text'
	Source *Source `json:"source,omitempty"` // if Type = 'image'
}

type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // e.g. 16000
}

type AnthropicClaudeMessagesInferenceParameters struct {
	AnthropicVersion string          `json:"anthropic_version"`
	Messages         []Message       `json:"messages"`
	System           string          `json:"system,omitempty"`
	Temperature      float64         `json:"temperature"`
	MaxTokens        int             `json:"max_tokens"`
	TopP             float64         `json:"top_p"`
	TopK             int             `json:"top_k,omitempty"` // recommended for advanced use cases only; usually enough to just use temp
	StopSequences    []string        `json:"stop_sequences,omitempty"`
	Thinking         *ThinkingConfig `json:"thinking,omitempty"`
}

type PerformanceConfig struct {
	Latency string `json:"latency"` // “latency” : “standard | optimized”
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
	return &AnthropicClaudeMessagesInferenceParameters{
		AnthropicVersion: "bedrock-2023-05-31",
		Temperature:      1.0,
		TopP:             0.999,
		MaxTokens:        defaultMaxTokens,
		StopSequences:    []string{},
		Thinking:         nil, // Will be set explicitly if needed
	}
}

// type AnthropicClaudeResponseBody struct {
// 	Completion string `json:"completion"`
// 	Stop       string `json:"stop"`
// 	StopReason string `json:"stop_reason"`
// }
//
//
// func (r *AnthropicClaudeResponseBody) Text() string {
// 	return strings.TrimSpace(r.Completion)
// }

type AnthropicClaudeStreamingChunk struct {
	Completion string `json:"completion"`
	Stop       string `json:"stop"`
	StopReason string `json:"stop_reason"`
}

type AnthropicClaudeMessagesResponse struct {
	// Type can be e.g.
	//   "message_start", "content_block_start", "content_block_delta", "content_block_delta",
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

	// type: "content_block_start"
	ContentBlock *struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content_block,omitempty"`

	// type: "content_block_delta"
	Delta *struct {
		StopReason   string `json:"stop_reason,omitempty"`
		StopSequence any    `json:"stop_sequence,omitempty"`
		Text         string `json:"text,omitempty"`
		Type         string `json:"type,omitempty"`
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
