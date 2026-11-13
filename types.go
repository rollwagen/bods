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
	ClaudeV45Opus
	ClaudeV46Opus
	ClaudeV47Opus
	ClaudeV46Sonnet
	ClaudeV48Opus
	ClaudeV5Fable
	ClaudeV5Sonnet
	ClaudeV5Opus
	ClaudeV55Opus
	ClaudeV55Sonnet
	ClaudeV55Haiku
	ClaudeV52Fable
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

// SourceTypeBase64 is the Source.Type value for base64-encoded image/document data.
const SourceTypeBase64 = "base64"

// SourceTypeText is the Source.Type value for inline plain-text document data.
const SourceTypeText = "text"

// CacheControlTypeEphemeral is the CacheControl.Type value for ephemeral prompt caching.
const CacheControlTypeEphemeral = "ephemeral"

// CacheControlMaxBreakpoints is the most cache_control markers a request may carry; a fifth
// returns a 400.
const CacheControlMaxBreakpoints = 4

// Effort levels for the output_config.effort parameter.
const (
	EffortMax    = "max"
	EffortXHigh  = "xhigh"
	EffortHigh   = "high"
	EffortMedium = "medium"
	EffortLow    = "low"
)

// For content type 'image', the following image formats exist
const (
	MessageContentTypeMediaTypeJPEG = "image/jpeg"
	MessageContentTypeMediaTypePNG  = "image/png"
	MessageContentTypeMediaTypeWEBP = "image/webp"
	MessageContentTypeMediaTypeGIF  = "image/gif"
	MessageContentTypeMediaTypePDF  = "application/pdf"
	MessageContentTypeMediaTypeText = "text/plain"
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
	if m == ClaudeV3Sonnet || m == ClaudeV3Haiku || m == ClaudeV3Opus || m == ClaudeV35Sonnet || m == ClaudeV35SonnetV2 || m == ClaudeV37Sonnet || m == ClaudeV4Sonnet || m == ClaudeV4Opus || m == ClaudeV45Sonnet || m == ClaudeV45Haiku || m == ClaudeV45Opus || m == ClaudeV46Opus || m == ClaudeV47Opus || m == ClaudeV46Sonnet || m == ClaudeV48Opus || m == ClaudeV5Fable || m == ClaudeV5Sonnet || m == ClaudeV5Opus || m == ClaudeV55Opus || m == ClaudeV55Sonnet || m == ClaudeV55Haiku || m == ClaudeV52Fable {
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
		ClaudeV45Opus.String(),
		ClaudeV46Opus.String(),
		ClaudeV47Opus.String(),
		ClaudeV46Sonnet.String(),
		ClaudeV48Opus.String(),
		ClaudeV5Fable.String(),
		ClaudeV5Sonnet.String(),
		ClaudeV5Opus.String(),
		ClaudeV55Opus.String(),
		ClaudeV55Sonnet.String(),
		ClaudeV55Haiku.String(),
		ClaudeV52Fable.String(),
	}
	modelID := normalizeToModelID(id)
	return slices.Contains(v3IDs, modelID)
}

func IsVisionCapable(id string) bool {
	modelID := normalizeToModelID(id)
	return IsClaude3OrHigherModelID(modelID) && modelID != ClaudeV35Haiku.String()
}

// IsCitationsSupported returns true if the given model ID supports citations.
// All Claude 3+ models support citations except Claude 3 Haiku.
func IsCitationsSupported(id string) bool {
	modelID := normalizeToModelID(id)
	return IsClaude3OrHigherModelID(modelID) && modelID != ClaudeV3Haiku.String()
}

// IsPromptCachingSupported returns true if the given model ID supports prompt caching.
// Prompt caching is generally available with Claude 3.7 Sonnet, Claude 3.5 Haiku, Claude 4, Claude 4.5, Claude 4.6, Claude 4.7, Claude 4.8, Claude Fable 5, Claude Sonnet 5, Claude Opus 5, Claude Opus 5.5, Claude Sonnet 5.5, Claude Haiku 5.5, and Claude Fable 5.2.
// Claude Opus 5.5, Claude Sonnet 5.5, Claude Haiku 5.5, and Claude Fable 5.2 have a 512-token minimum
// cacheable prompt (see opus_5_5.md and sonnet_5_5.md, "Feature support", haiku55.md, "Good to know", and
// fable52.md, "Lower prompt caching minimum"); Claude Sonnet 5's is 1,024, and Claude Haiku 4.5's is 4,096.
// See: https://docs.aws.amazon.com/bedrock/latest/userguide/prompt-caching.html#prompt-caching-models
func IsPromptCachingSupported(id string) bool {
	modelID := normalizeToModelID(id)
	cachingSupportedModels := []string{
		ClaudeV35Haiku.String(),  // Claude 3.5 Haiku
		ClaudeV37Sonnet.String(), // Claude 3.7 Sonnet
		ClaudeV4Sonnet.String(),  // Claude 4 Sonnet
		ClaudeV4Opus.String(),    // Claude 4 Opus
		ClaudeV45Sonnet.String(), // Claude 4.5 Sonnet
		ClaudeV45Opus.String(),   // Claude 4.5 Opus
		ClaudeV45Haiku.String(),  // Claude 4.5 Haiku
		ClaudeV46Opus.String(),   // Claude 4.6 Opus
		ClaudeV47Opus.String(),   // Claude 4.7 Opus
		ClaudeV46Sonnet.String(), // Claude 4.6 Sonnet
		ClaudeV48Opus.String(),   // Claude 4.8 Opus
		ClaudeV5Fable.String(),   // Claude Fable 5
		ClaudeV5Sonnet.String(),  // Claude Sonnet 5
		ClaudeV5Opus.String(),    // Claude Opus 5
		ClaudeV55Opus.String(),   // Claude Opus 5.5
		ClaudeV55Sonnet.String(), // Claude Sonnet 5.5
		ClaudeV55Haiku.String(),  // Claude Haiku 5.5
		ClaudeV52Fable.String(),  // Claude Fable 5.2
	}
	return slices.Contains(cachingSupportedModels, modelID)
}

// withCacheBreakpoints returns the messages to send, carrying at most CacheControlMaxBreakpoints
// cache_control markers. With markTail it also marks the last block of the last message: in the
// text editor tool loop each turn then writes the conversation so far to the cache, and the next
// turn reads it back rather than paying full input price for every file viewed. The marker moves
// forward each turn, which doesn't invalidate preserved thinking blocks: seen on Opus 5.5, and
// on Fable 5.2, whose prefix check returns a 400 on any edit (verified live 2026-11-13).
//
// Surplus explicit markers (every PDF and pasteboard file gets one) are dropped earliest first,
// since a later marker caches everything before it as well.
//
// msgs is left unchanged: a tail marker stored in the history would stay on every later turn
// and use up the slots. Only the Content slices that change are copied.
func withCacheBreakpoints(msgs []Message, markTail bool) []Message {
	out := slices.Clone(msgs)

	tail := -1 // index of the block to mark in the last message
	if markTail && len(out) > 0 {
		last := out[len(out)-1].Content
		if n := len(last); n > 0 && last[n-1].Type != MessageContentTypeThinking {
			tail = n - 1
		}
	}
	isTail := func(i, j int) bool { return tail >= 0 && i == len(out)-1 && j == tail }

	excess := -CacheControlMaxBreakpoints
	if tail >= 0 {
		excess++
	}
	for i := range out {
		for j, c := range out[i].Content {
			if c.CacheControl != nil && !isTail(i, j) {
				excess++
			}
		}
	}

	copied := make([]bool, len(out))
	block := func(i, j int) *Content {
		if !copied[i] {
			out[i].Content = slices.Clone(out[i].Content)
			copied[i] = true
		}
		return &out[i].Content[j]
	}

	for i := 0; i < len(out) && excess > 0; i++ {
		for j := 0; j < len(out[i].Content) && excess > 0; j++ {
			if out[i].Content[j].CacheControl != nil && !isTail(i, j) {
				block(i, j).CacheControl = nil
				excess--
			}
		}
	}

	if tail >= 0 {
		block(len(out)-1, tail).CacheControl = &CacheControl{Type: CacheControlTypeEphemeral}
	}
	return out
}

// IsEffortParamSupported returns true if the given model ID supports the effort parameter.
// The effort parameter is supported by Claude Opus 4.5, Claude Opus 4.6, Claude Opus 4.7, Claude Opus 4.8,
// Claude Sonnet 4.6 (which defaults to effort "high"), Claude Fable 5, Claude Sonnet 5, Claude Opus 5,
// Claude Opus 5.5, Claude Sonnet 5.5, Claude Haiku 5.5 (the first Haiku with effort), and Claude Fable 5.2.
// Note that "xhigh"/"max" are Opus/Fable-only among older models, but Claude Sonnet 5, Claude Opus 5,
// Claude Opus 5.5, Claude Sonnet 5.5, Claude Haiku 5.5, and Claude Fable 5.2 accept the full range
// ("low"/"medium"/"high"/"xhigh"/"max"); Sonnet 4.6 still caps at "high"/"medium"/"low".
// On Claude Opus 5.5 effort is the *only* thinking control, and a request that omits it runs at "medium"
// where Claude Opus 5 ran at "high" - so pass --effort explicitly to get the old depth back. Opus 5.5 also
// thinks more per turn than Opus 5 at the same effort level, most of all at "xhigh" and "max".
// Claude Sonnet 5.5 defaults to "high", but its levels are recalibrated: a level doesn't produce the
// same amount of thinking as on Sonnet 5, so templates set effort explicitly.
// Claude Haiku 5.5 runs at "medium" when effort is omitted, like Opus 5.5. Its doc doesn't list the levels
// (thinking can be turned off "at `high` effort or below" implies "xhigh" and "max"), but all five were
// accepted live on Bedrock on 2026-11-12.
// Claude Fable 5.2 defaults to "high", one level above Opus 5.5, and at "high" and above it thinks before
// almost every reply.
// See: opus47vision.md ("Migrating to Claude Sonnet 4.6"), fable.md, whats-new-sonnet5.md, opus5.md,
// opus_5_5.md ("The default effort is `medium`"), sonnet_5_5.md ("Behavior differences"),
// haiku55.md ("How it compares", "Adaptive thinking and effort"), and fable52.md ("Feature support").
func IsEffortParamSupported(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV45Opus.String() || modelID == ClaudeV46Opus.String() || modelID == ClaudeV47Opus.String() || modelID == ClaudeV46Sonnet.String() || modelID == ClaudeV48Opus.String() || modelID == ClaudeV5Fable.String() || modelID == ClaudeV5Sonnet.String() || modelID == ClaudeV5Opus.String() || modelID == ClaudeV55Opus.String() || modelID == ClaudeV55Sonnet.String() || modelID == ClaudeV55Haiku.String() || modelID == ClaudeV52Fable.String()
}

// IsSamplingParamsRejected returns true for models that reject ANY non-default
// sampling parameter (temperature, top_p, top_k) with a 400 error. This is the
// case from Claude Opus 4.7 onwards, for Claude Fable 5, Claude Sonnet 5, Claude Opus 5, Claude Opus 5.5,
// Claude Sonnet 5.5, Claude Haiku 5.5, and Claude Fable 5.2.
// For these models all sampling parameters must be omitted from the request entirely.
// See: opus47vision.md ("Sampling parameters removed"), fable.md ("`top_p` is deprecated for this model"),
// whats-new-sonnet5.md ("Sampling parameters not accepted"), opus5.md ("Sampling parameters removed"),
// sonnet_5_5.md ("Good to know"), haiku55.md ("Remove sampling parameters"), and fable52.md
// ("Good to know").
func IsSamplingParamsRejected(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV47Opus.String() || modelID == ClaudeV48Opus.String() || modelID == ClaudeV5Fable.String() || modelID == ClaudeV5Sonnet.String() || modelID == ClaudeV5Opus.String() || modelID == ClaudeV55Opus.String() || modelID == ClaudeV55Sonnet.String() || modelID == ClaudeV55Haiku.String() || modelID == ClaudeV52Fable.String()
}

// IsClaude45OrHigherModel returns true if the given model ID is Claude 4.5+ (Sonnet, Haiku, Opus, or Opus 4.6).
// Claude 4.5+ models have a breaking change where only temperature OR top_p can be specified, not both.
func IsClaude45OrHigherModel(id string) bool {
	modelID := normalizeToModelID(id)
	claude45PlusModels := []string{
		ClaudeV45Sonnet.String(), // Claude 4.5 Sonnet
		ClaudeV45Haiku.String(),  // Claude 4.5 Haiku
		ClaudeV45Opus.String(),   // Claude 4.5 Opus
		ClaudeV46Opus.String(),   // Claude 4.6 Opus
		ClaudeV47Opus.String(),   // Claude 4.7 Opus
		ClaudeV46Sonnet.String(), // Claude 4.6 Sonnet
		ClaudeV48Opus.String(),   // Claude 4.8 Opus
		ClaudeV5Fable.String(),   // Claude Fable 5
		ClaudeV5Sonnet.String(),  // Claude Sonnet 5
		ClaudeV5Opus.String(),    // Claude Opus 5
		ClaudeV55Opus.String(),   // Claude Opus 5.5
		ClaudeV55Sonnet.String(), // Claude Sonnet 5.5
		ClaudeV55Haiku.String(),  // Claude Haiku 5.5
		ClaudeV52Fable.String(),  // Claude Fable 5.2
	}
	return slices.Contains(claude45PlusModels, modelID)
}

// IsOpus46Model returns true if the given model ID is Claude Opus 4.6.
func IsOpus46Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV46Opus.String()
}

// IsOpus47Model returns true if the given model ID is Claude Opus 4.7.
func IsOpus47Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV47Opus.String()
}

// IsOpus48Model returns true if the given model ID is Claude Opus 4.8.
func IsOpus48Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV48Opus.String()
}

// IsFable5Model returns true if the given model ID is Claude Fable 5.
func IsFable5Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV5Fable.String()
}

// IsSonnet5Model returns true if the given model ID is Claude Sonnet 5.
func IsSonnet5Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV5Sonnet.String()
}

// IsOpus5Model returns true if the given model ID is Claude Opus 5.
func IsOpus5Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV5Opus.String()
}

// IsOpus55Model returns true if the given model ID is Claude Opus 5.5.
func IsOpus55Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV55Opus.String()
}

// IsSonnet55Model returns true if the given model ID is Claude Sonnet 5.5.
func IsSonnet55Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV55Sonnet.String()
}

// IsHaiku55Model returns true if the given model ID is Claude Haiku 5.5.
func IsHaiku55Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV55Haiku.String()
}

// IsFable52Model returns true if the given model ID is Claude Fable 5.2.
func IsFable52Model(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV52Fable.String()
}

// IsInferenceProfileOnlyModel returns true for models that Amazon Bedrock serves only through an
// inference profile (global. or us.), so an invoke by the bare model ID fails. Only Claude Fable
// 5.2 so far. Other models may be profile-only too; add one only when its doc says so.
// See: fable52.md ("Availability", "Update your model name").
func IsInferenceProfileOnlyModel(id string) bool {
	return normalizeToModelID(id) == ClaudeV52Fable.String()
}

// IsAdaptiveThinkingModel returns true for models that use adaptive thinking
// rather than manual budget_tokens (Opus 4.6, Opus 4.7, Opus 4.8, Sonnet 4.6,
// Fable 5, Sonnet 5, Opus 5, Opus 5.5, Sonnet 5.5, Haiku 5.5, and Fable 5.2). On Fable 5, Opus 5.5,
// and Fable 5.2 adaptive thinking is always on and cannot be disabled; on Sonnet 5 and Opus 5 it is on
// by default. Sonnet 5.5 runs it by default too, and its lowest setting is
// thinking:{"type":"between_tools"} (low to high effort only), which bods doesn't send. Haiku 5.5 also
// thinks by default; it accepts thinking:{"type":"disabled"} at "high" effort or below, which bods
// doesn't send either.
//
// This predicate is what keeps a manual thinking config off the wire: Opus 5.5, Sonnet 5.5, and Fable 5.2
// reject both thinking:{"type":"disabled"} and thinking:{"type":"enabled","budget_tokens":N} with a 400,
// and Haiku 5.5 rejects the latter. startMessagesCmd only builds it for models this returns false for.
// startMessagesCmd also uses it to refuse assistant prefill, which these models reject with a 400
// (Haiku 5.5 even with thinking off).
// See: opus_5_5.md ("Thinking can't be disabled"), sonnet_5_5.md ("Turn off up-front thinking"),
// haiku55.md ("Configure thinking", "Replace assistant prefill"), and fable52.md ("Thinking and tool choice").
func IsAdaptiveThinkingModel(id string) bool {
	modelID := normalizeToModelID(id)
	return modelID == ClaudeV46Opus.String() || modelID == ClaudeV47Opus.String() || modelID == ClaudeV46Sonnet.String() || modelID == ClaudeV48Opus.String() || modelID == ClaudeV5Fable.String() || modelID == ClaudeV5Sonnet.String() || modelID == ClaudeV5Opus.String() || modelID == ClaudeV55Opus.String() || modelID == ClaudeV55Sonnet.String() || modelID == ClaudeV55Haiku.String() || modelID == ClaudeV52Fable.String()
}

// maxOutputTokens is the largest max_tokens each model accepts without a beta header. bods
// uses it as the default output budget: adaptive-thinking models spend part of every turn
// thinking, and a text editor turn that writes a file needs room for the whole file on top of
// that, so a smaller default ends turns mid tool call. Claude 3.7 Sonnet reaches 128K and
// Claude 3.5 Sonnet (June 2024) 8192 only with beta headers bods doesn't send.
// See: opus_5_5.md ("128k max output tokens"), haiku55.md and fable52.md ("Capabilities"), and
// Anthropic's models overview.
var maxOutputTokens = map[string]int{
	ClaudeV3Sonnet.String():    4096,
	ClaudeV3Haiku.String():     4096,
	ClaudeV3Opus.String():      4096,
	ClaudeV35Sonnet.String():   4096,
	ClaudeV35SonnetV2.String(): 8192,
	ClaudeV35Haiku.String():    8192,
	ClaudeV37Sonnet.String():   64000,
	ClaudeV4Sonnet.String():    64000,
	ClaudeV4Opus.String():      32000,
	ClaudeV45Sonnet.String():   64000,
	ClaudeV45Haiku.String():    64000,
	ClaudeV45Opus.String():     64000,
	ClaudeV46Opus.String():     128000,
	ClaudeV47Opus.String():     128000,
	ClaudeV46Sonnet.String():   128000,
	ClaudeV48Opus.String():     128000,
	ClaudeV5Fable.String():     128000,
	ClaudeV5Sonnet.String():    128000,
	ClaudeV5Opus.String():      128000,
	ClaudeV55Opus.String():     128000,
	ClaudeV55Sonnet.String():   128000,
	ClaudeV55Haiku.String():    128000,
	ClaudeV52Fable.String():    128000,
}

// MaxOutputTokens returns the largest max_tokens the given model accepts, and whether bods
// knows the model. For an unknown model it returns defaultMaxTokens.
func MaxOutputTokens(id string) (int, bool) {
	if n, ok := maxOutputTokens[normalizeToModelID(id)]; ok {
		return n, true
	}
	return defaultMaxTokens, false
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
	case ClaudeV45Opus:
		return "anthropic.claude-opus-4-5-20251101-v1:0"
	case ClaudeV46Opus:
		return "anthropic.claude-opus-4-6-v1"
	case ClaudeV47Opus:
		return "anthropic.claude-opus-4-7"
	case ClaudeV46Sonnet:
		return "anthropic.claude-sonnet-4-6"
	case ClaudeV48Opus:
		return "anthropic.claude-opus-4-8"
	case ClaudeV5Fable:
		return "anthropic.claude-fable-5"
	case ClaudeV5Sonnet:
		return "anthropic.claude-sonnet-5"
	case ClaudeV5Opus:
		return "anthropic.claude-opus-5"
	case ClaudeV55Opus:
		return "anthropic.claude-opus-5-5"
	case ClaudeV55Sonnet:
		return "anthropic.claude-sonnet-5-5"
	case ClaudeV55Haiku:
		return "anthropic.claude-haiku-5-5"
	case ClaudeV52Fable:
		return "anthropic.claude-fable-5-2"
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
	ClaudeV45Opus.String(),
	ClaudeV46Opus.String(),
	ClaudeV47Opus.String(),
	ClaudeV46Sonnet.String(),
	ClaudeV48Opus.String(),
	ClaudeV5Fable.String(),
	ClaudeV5Sonnet.String(),
	ClaudeV5Opus.String(),
	ClaudeV55Opus.String(),
	ClaudeV55Sonnet.String(),
	ClaudeV55Haiku.String(),
	ClaudeV52Fable.String(),
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

type CitationResponse struct {
	Type            string `json:"type"` // "char_location", "page_location", "content_block_location"
	CitedText       string `json:"cited_text"`
	DocumentIndex   int    `json:"document_index"`
	DocumentTitle   string `json:"document_title,omitempty"`
	StartCharIndex  int    `json:"start_char_index,omitempty"`
	EndCharIndex    int    `json:"end_char_index,omitempty"`
	StartPageNumber int    `json:"start_page_number,omitempty"`
	EndPageNumber   int    `json:"end_page_number,omitempty"`
	StartBlockIndex int    `json:"start_block_index,omitempty"`
	EndBlockIndex   int    `json:"end_block_index,omitempty"`
}

// StopDetails accompanies stop_reason "refusal" and names the policy area that declined the
// request. Category is e.g. "cyber", "bio", or "reasoning_extraction"; Claude Opus 5.5 runs more
// safeguard classifiers than Claude Opus 5, so more categories can appear.
// See: opus_5_5.md ("Refusals and fallback").
type StopDetails struct {
	Type     string `json:"type,omitempty"`
	Category string `json:"category,omitempty"`
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

// MarshalJSON forces the "thinking" field to be present for thinking blocks,
// even when empty. Adaptive-thinking models (Sonnet 5, Opus 4.7+, Fable 5)
// return thinking blocks with an empty "thinking" field and a signature; the
// block must be round-tripped unchanged alongside tool_result, but the struct's
// omitempty tag would drop the empty field and Bedrock rejects it with
// "thinking.thinking: Field required". Other content types keep omitempty.
func (c Content) MarshalJSON() ([]byte, error) {
	type alias Content
	if c.Type == MessageContentTypeThinking {
		// The outer Thinking (no omitempty) shadows the embedded alias's
		// json:"thinking,omitempty" field, so the key is always emitted.
		return json.Marshal(&struct {
			Thinking string `json:"thinking"`
			*alias
		}{Thinking: c.Thinking, alias: (*alias)(&c)})
	}
	return json.Marshal((alias)(c))
}

type ThinkingConfig struct {
	Type         string `json:"type"`                    // "enabled" or "adaptive"
	BudgetTokens int    `json:"budget_tokens,omitempty"` // budget_tokens is 1024 tokens (omitted for adaptive thinking)
	Display      string `json:"display,omitempty"`       // "summarized" to receive readable thinking summaries; API default is "omitted"
}

type OutputConfig struct {
	Effort string `json:"effort"` // "max" | "xhigh" | "high" | "medium" | "low"
}

type AnthropicClaudeMessagesInferenceParameters struct {
	AnthropicVersion string          `json:"anthropic_version"`
	Messages         []Message       `json:"messages"`
	System           string          `json:"system,omitempty"`
	Temperature      *float64        `json:"temperature,omitempty"` // pointer allows omitting for models that reject sampling params (Opus 4.7+)
	MaxTokens        int             `json:"max_tokens"`
	TopP             *float64        `json:"top_p,omitempty"` // pointer allows omitting for Claude 4.5 models
	TopK             int             `json:"top_k,omitempty"` // recommended for advanced use cases only; usually enough to just use temp
	StopSequences    []string        `json:"stop_sequences,omitempty"`
	Thinking         *ThinkingConfig `json:"thinking,omitempty"`
	OutputConfig     *OutputConfig   `json:"output_config,omitempty"`
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

func NewAdaptiveThinkingConfig() *ThinkingConfig {
	// Adaptive thinking models (Opus 4.6+, Fable 5) default thinking.display to
	// "omitted", returning no readable chain of thought. This config is only
	// built when --think is set (see startMessagesCmd), so requesting
	// "summarized" here surfaces thinking summaries for users who opted in,
	// while the default (no --think) stays quiet.
	return &ThinkingConfig{
		Type:    "adaptive",
		Display: "summarized",
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
	temperature := 1.0
	return &AnthropicClaudeMessagesInferenceParameters{
		AnthropicVersion: "bedrock-2023-05-31",
		Temperature:      &temperature, // pointer to allow omitting for models that reject sampling params (Opus 4.7+)
		TopP:             &topP,        // pointer to allow omitting for Claude 4.5 models
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
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
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
		StopReason   string            `json:"stop_reason,omitempty"`
		StopSequence any               `json:"stop_sequence,omitempty"`
		StopDetails  *StopDetails      `json:"stop_details,omitempty"` // present with stop_reason "refusal"
		Type         string            `json:"type,omitempty"`
		Text         string            `json:"text,omitempty"`
		Thinking     string            `json:"thinking,omitempty"`
		PartialJSON  string            `json:"partial_json,omitempty"`
		Signature    string            `json:"signature,omitempty"`
		Citation     *CitationResponse `json:"citation,omitempty"`
	} `json:"delta,omitempty"`

	// type: "message_delta" - some responses carry stop_details alongside the delta rather than
	// inside it, so decode both and prefer whichever is populated.
	StopDetails *StopDetails `json:"stop_details,omitempty"`

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
