package main

type AnthropicModel int

const (
	UndefinedAnthropicModel AnthropicModel = iota
	ClaudeV2
	ClaudeV21
	ClaudeV3Sonnet
	ClaudeV3Haiku
)

func (m AnthropicModel) String() string {
	switch m {
	case ClaudeV2:
		return "anthropic.claude-v2"
	case ClaudeV21:
		return "anthropic.claude-v2:1"
	case ClaudeV3Sonnet:
		return "anthropic.claude-3-sonnet-20240229-v1:0"
	case ClaudeV3Haiku:
		return "anthropic.claude-3-haiku-20240307-v1:0"
	default:
		panic("AnthropicModel String()  - unhandled default case")
	}
}

var AnthrophicModelsIDs = []string{
	ClaudeV2.String(),
	ClaudeV21.String(),
	ClaudeV3Sonnet.String(),
	ClaudeV3Haiku.String(),
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
	Content string `json:"content"`
	Role    string `json:"role"` // "user" or "assistnat"
}

type AnthropicClaudeMessagesInferenceParameters struct {
	AnthropicVersion string    `json:"anthropic_version"`
	Messages         []Message `json:"messages"`
	System           string    `json:"system,omitempty"`
	Temperature      float64   `json:"temperature"`
	MaxTokens        int       `json:"max_tokens"`
	TopP             float64   `json:"top_p"`
	TopK             int       `json:"top_k,omitempty"` // recommended for advanced use cases only; usually enough to just use temp
	StopSequences    []string  `json:"stop_sequences,omitempty"`
}

func NewAnthropicClaudeInferenceParameters() *AnthropicClaudeInferenceParameters {
	return &AnthropicClaudeInferenceParameters{
		Temperature:   1.0,
		TopP:          0.999,
		TopK:          250,
		MaxTokens:     200,
		StopSequences: []string{},
	}
}

func NewAnthropicClaudeMessagesInferenceParameters() *AnthropicClaudeMessagesInferenceParameters {
	return &AnthropicClaudeMessagesInferenceParameters{
		AnthropicVersion: "bedrock-2023-05-31",
		Temperature:      1.0,
		TopP:             0.999,
		MaxTokens:        200,
		StopSequences:    []string{},
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
	UndefinedEventTyp ResponseEventType = iota
	MessageStart                        // message_start
	ContentBlockStart                   // content_block_start
	Ping                                // ping
	ContentBlockDelta                   // content_block_delta
	ContentBlockStop                    // content_block_stop
	MessageDelta                        // message_delta
	MessageStop                         // message_stop
)

func (r ResponseEventType) String() string {
	switch r {
	case MessageStart:
		return "message_start"
	case ContentBlockStart:
		return "content_block_start"
	case Ping:
		return "ping"
	case ContentBlockDelta:
		return "content_block_delta"
	case ContentBlockStop:
		return "content_block_stop"
	case MessageDelta:
		return "message_delta"
	case MessageStop:
		return "message_stop"
	default:
		return "unknown ResponseEventType"
	}
}
