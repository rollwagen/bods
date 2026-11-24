package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const (
	nameClaude47Opus   = "Claude 4.7 Opus"
	nameClaude48Opus   = "Claude 4.8 Opus"
	nameClaudeFable5   = "Claude Fable 5"
	nameClaudeSonnet5  = "Claude Sonnet 5"
	nameClaudeOpus5    = "Claude Opus 5"
	nameClaudeOpus55   = "Claude Opus 5.5"
	nameClaudeSonnet55 = "Claude Sonnet 5.5"
	nameClaudeHaiku55  = "Claude Haiku 5.5"
	nameClaudeFable52  = "Claude Fable 5.2"
)

func TestIsVisionCapable(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     "Claude 3 Sonnet",
			modelID:  ClaudeV3Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 3.5 Haiku (No Vision)",
			modelID:  ClaudeV35Haiku.String(),
			expected: false,
		},
		{
			name:     "Claude 4.5 Opus",
			modelID:  ClaudeV45Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Opus (Raw String)",
			modelID:  "anthropic.claude-opus-4-5-20251101-v1:0",
			expected: true,
		},
		{
			name:     "Claude 4.6 Opus",
			modelID:  ClaudeV46Opus.String(),
			expected: true,
		},
		{
			name:     nameClaude47Opus,
			modelID:  ClaudeV47Opus.String(),
			expected: true,
		},
		{
			name:     nameClaude48Opus,
			modelID:  ClaudeV48Opus.String(),
			expected: true,
		},
		{
			name:     nameClaudeFable5,
			modelID:  ClaudeV5Fable.String(),
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-sonnet-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVisionCapable(tt.modelID); got != tt.expected {
				t.Errorf("IsVisionCapable(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsPromptCachingSupported(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     "Claude 3.5 Haiku",
			modelID:  ClaudeV35Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude 3.7 Sonnet",
			modelID:  ClaudeV37Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4 Sonnet",
			modelID:  ClaudeV4Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4 Opus",
			modelID:  ClaudeV4Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Sonnet",
			modelID:  ClaudeV45Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Opus",
			modelID:  ClaudeV45Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Haiku",
			modelID:  ClaudeV45Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude 4.6 Opus",
			modelID:  ClaudeV46Opus.String(),
			expected: true,
		},
		{
			name:     nameClaude47Opus,
			modelID:  ClaudeV47Opus.String(),
			expected: true,
		},
		{
			name:     nameClaude48Opus,
			modelID:  ClaudeV48Opus.String(),
			expected: true,
		},
		{
			name:     nameClaudeFable5,
			modelID:  ClaudeV5Fable.String(),
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-sonnet-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
		{
			name:     "Claude 3 Sonnet (No Caching)",
			modelID:  ClaudeV3Sonnet.String(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPromptCachingSupported(tt.modelID); got != tt.expected {
				t.Errorf("IsPromptCachingSupported(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsSamplingParamsRejected(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     nameClaude47Opus,
			modelID:  ClaudeV47Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.7 Opus (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-4-7",
			expected: true,
		},
		{
			name:     nameClaude48Opus,
			modelID:  ClaudeV48Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.8 Opus (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-4-8",
			expected: true,
		},
		{
			name:     nameClaudeFable5,
			modelID:  ClaudeV5Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-fable-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-sonnet-5",
			expected: true,
		},
		{
			name:     "Claude 4.6 Opus (sampling still allowed)",
			modelID:  ClaudeV46Opus.String(),
			expected: false,
		},
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
		{
			name:     "Claude 4.6 Sonnet (temperature OR top_p only, not rejected outright)",
			modelID:  ClaudeV46Sonnet.String(),
			expected: false,
		},
		{
			name:     "Claude 4.5 Opus (sampling still allowed)",
			modelID:  ClaudeV45Opus.String(),
			expected: false,
		},
		{
			name:     "Claude 3.7 Sonnet",
			modelID:  ClaudeV37Sonnet.String(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSamplingParamsRejected(tt.modelID); got != tt.expected {
				t.Errorf("IsSamplingParamsRejected(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsCitationsSupported(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     "Claude 3 Haiku (No Citations)",
			modelID:  ClaudeV3Haiku.String(),
			expected: false,
		},
		{
			name:     "Claude 3 Sonnet",
			modelID:  ClaudeV3Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 3.5 Sonnet v2",
			modelID:  ClaudeV35SonnetV2.String(),
			expected: true,
		},
		{
			name:     "Claude 3.7 Sonnet",
			modelID:  ClaudeV37Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4 Sonnet",
			modelID:  ClaudeV4Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4.6 Opus",
			modelID:  ClaudeV46Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.6 Sonnet",
			modelID:  ClaudeV46Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude 4.7 Opus",
			modelID:  ClaudeV47Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.8 Opus",
			modelID:  ClaudeV48Opus.String(),
			expected: true,
		},
		{
			name:     nameClaudeFable5,
			modelID:  ClaudeV5Fable.String(),
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-sonnet-5",
			expected: true,
		},
		{
			name:     "Claude 4.5 Opus with prefix",
			modelID:  "eu.anthropic.claude-opus-4-5-20251101-v1:0",
			expected: true,
		},
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCitationsSupported(tt.modelID); got != tt.expected {
				t.Errorf("IsCitationsSupported(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsEffortParamSupported(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     nameClaude48Opus,
			modelID:  ClaudeV48Opus.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Sonnet (No Effort)",
			modelID:  ClaudeV45Sonnet.String(),
			expected: false,
		},
		{
			name:     "Claude 3.7 Sonnet (No Effort)",
			modelID:  ClaudeV37Sonnet.String(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEffortParamSupported(tt.modelID); got != tt.expected {
				t.Errorf("IsEffortParamSupported(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsAdaptiveThinkingModel(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		expected bool
	}{
		{
			name:     nameClaudeOpus5,
			modelID:  ClaudeV5Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5",
			expected: true,
		},
		{
			name:     nameClaudeOpus55,
			modelID:  ClaudeV55Opus.String(),
			expected: true,
		},
		{
			name:     "Claude Opus 5.5 (region-prefixed)",
			modelID:  "eu.anthropic.claude-opus-5-5",
			expected: true,
		},
		{
			name:     nameClaudeSonnet55,
			modelID:  ClaudeV55Sonnet.String(),
			expected: true,
		},
		{
			name:     "Claude Sonnet 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-sonnet-5-5",
			expected: true,
		},
		{
			name:     nameClaudeHaiku55,
			modelID:  ClaudeV55Haiku.String(),
			expected: true,
		},
		{
			name:     "Claude Haiku 5.5 (global inference profile)",
			modelID:  "global.anthropic.claude-haiku-5-5",
			expected: true,
		},
		{
			name:     nameClaudeFable52,
			modelID:  ClaudeV52Fable.String(),
			expected: true,
		},
		{
			name:     "Claude Fable 5.2 (global inference profile)",
			modelID:  "global.anthropic.claude-fable-5-2",
			expected: true,
		},
		{
			name:     nameClaudeSonnet5,
			modelID:  ClaudeV5Sonnet.String(),
			expected: true,
		},
		{
			name:     nameClaudeFable5,
			modelID:  ClaudeV5Fable.String(),
			expected: true,
		},
		{
			name:     "Claude 4.5 Opus (manual budget_tokens)",
			modelID:  ClaudeV45Opus.String(),
			expected: false,
		},
		{
			name:     "Claude 3.7 Sonnet (manual budget_tokens)",
			modelID:  ClaudeV37Sonnet.String(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAdaptiveThinkingModel(tt.modelID); got != tt.expected {
				t.Errorf("IsAdaptiveThinkingModel(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestContentMarshalJSON(t *testing.T) {
	t.Run("thinking block keeps empty thinking field", func(t *testing.T) {
		c := Content{Type: MessageContentTypeThinking, Thinking: "", Signature: "abc"}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		got := string(b)
		if !strings.Contains(got, `"thinking":""`) {
			t.Errorf("expected empty thinking field to be present, got %s", got)
		}
		if !strings.Contains(got, `"signature":"abc"`) {
			t.Errorf("expected signature to be present, got %s", got)
		}
	})

	t.Run("thinking block keeps non-empty thinking field", func(t *testing.T) {
		c := Content{Type: MessageContentTypeThinking, Thinking: "reasoning", Signature: "sig"}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if got := string(b); !strings.Contains(got, `"thinking":"reasoning"`) {
			t.Errorf("expected thinking text to be present, got %s", got)
		}
	})

	t.Run("non-thinking block omits thinking field", func(t *testing.T) {
		c := Content{Type: MessageContentTypeText, Text: "hi"}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if got := string(b); strings.Contains(got, "thinking") {
			t.Errorf("expected no thinking field for text block, got %s", got)
		}
	})

	t.Run("tool_use block round-trips Input unchanged", func(t *testing.T) {
		input := json.RawMessage(`{"command":"view","path":"/tmp/x"}`)
		c := Content{Type: MessageContentTypeToolUse, ID: "toolu_1", Name: "str_replace_based_edit_tool", Input: input}
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if got := string(b); !strings.Contains(got, `"input":{"command":"view","path":"/tmp/x"}`) {
			t.Errorf("expected input to round-trip unchanged, got %s", got)
		}
	})
}

// TestStopDetailsUnmarshal covers decoding the message_delta event that carries a refusal. The
// streaming loop reads stop_details from either position, so both shapes must decode.
func TestStopDetailsUnmarshal(t *testing.T) {
	tests := []struct {
		name             string
		event            string
		expectedReason   string
		expectedCategory string
	}{
		{
			name:             "refusal with stop_details inside the delta",
			event:            `{"type":"message_delta","delta":{"stop_reason":"refusal","stop_sequence":null,"stop_details":{"type":"refusal","category":"bio"}},"usage":{"output_tokens":12}}`,
			expectedReason:   "refusal",
			expectedCategory: "bio",
		},
		{
			name:             "refusal with stop_details beside the delta",
			event:            `{"type":"message_delta","delta":{"stop_reason":"refusal"},"stop_details":{"type":"refusal","category":"reasoning_extraction"}}`,
			expectedReason:   "refusal",
			expectedCategory: "reasoning_extraction",
		},
		{
			name:           "ordinary end_turn carries no stop_details",
			event:          `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":42}}`,
			expectedReason: "end_turn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AnthropicClaudeMessagesResponse
			if err := json.Unmarshal([]byte(tt.event), &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if got.Delta == nil {
				t.Fatal("expected Delta to be populated")
			}
			if got.Delta.StopReason != tt.expectedReason {
				t.Errorf("StopReason = %q, want %q", got.Delta.StopReason, tt.expectedReason)
			}

			var category string
			switch {
			case got.Delta.StopDetails != nil:
				category = got.Delta.StopDetails.Category
			case got.StopDetails != nil:
				category = got.StopDetails.Category
			}
			if category != tt.expectedCategory {
				t.Errorf("stop_details category = %q, want %q", category, tt.expectedCategory)
			}
		})
	}
}

func TestIsInferenceProfileOnlyModel(t *testing.T) {
	tests := []struct {
		name    string
		modelID string
		want    bool
	}{
		{name: nameClaudeFable52, modelID: ClaudeV52Fable.String(), want: true},
		{name: "Claude Fable 5.2 (global inference profile)", modelID: "global.anthropic.claude-fable-5-2", want: true},
		{name: "Claude Fable 5.2 (US inference profile)", modelID: "us.anthropic.claude-fable-5-2", want: true},
		{name: nameClaudeFable5, modelID: ClaudeV5Fable.String(), want: false},
		{name: nameClaudeOpus55, modelID: ClaudeV55Opus.String(), want: false},
		{name: nameClaudeHaiku55, modelID: ClaudeV55Haiku.String(), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInferenceProfileOnlyModel(tt.modelID); got != tt.want {
				t.Errorf("IsInferenceProfileOnlyModel(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestMaxOutputTokens(t *testing.T) {
	// A model missing from the table would silently fall back to defaultMaxTokens, which is
	// too small for a text editor turn that writes a whole file.
	for _, id := range AnthrophicModelsIDs {
		if _, known := MaxOutputTokens(id); !known {
			t.Errorf("MaxOutputTokens(%q): no ceiling registered", id)
		}
	}

	tests := []struct {
		name      string
		modelID   string
		want      int
		wantKnown bool
	}{
		{name: nameClaudeOpus55, modelID: ClaudeV55Opus.String(), want: 128000, wantKnown: true},
		{name: "Claude Opus 5.5 (region-prefixed)", modelID: "eu.anthropic.claude-opus-5-5", want: 128000, wantKnown: true},
		{name: nameClaudeSonnet55, modelID: ClaudeV55Sonnet.String(), want: 128000, wantKnown: true},
		{name: "Claude Sonnet 5.5 (global inference profile)", modelID: "global.anthropic.claude-sonnet-5-5", want: 128000, wantKnown: true},
		{name: nameClaudeHaiku55, modelID: ClaudeV55Haiku.String(), want: 128000, wantKnown: true},
		{name: "Claude Haiku 5.5 (global inference profile)", modelID: "global.anthropic.claude-haiku-5-5", want: 128000, wantKnown: true},
		{name: nameClaudeFable52, modelID: ClaudeV52Fable.String(), want: 128000, wantKnown: true},
		{name: "Claude Fable 5.2 (global inference profile)", modelID: "global.anthropic.claude-fable-5-2", want: 128000, wantKnown: true},
		{name: "Claude 4 Opus", modelID: ClaudeV4Opus.String(), want: 32000, wantKnown: true},
		{name: "Claude 3.7 Sonnet (128K needs a beta header)", modelID: ClaudeV37Sonnet.String(), want: 64000, wantKnown: true},
		{name: "Claude 3 Haiku", modelID: ClaudeV3Haiku.String(), want: 4096, wantKnown: true},
		{name: "unknown model", modelID: "anthropic.claude-v2:1", want: defaultMaxTokens, wantKnown: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, known := MaxOutputTokens(tt.modelID)
			if got != tt.want || known != tt.wantKnown {
				t.Errorf("MaxOutputTokens(%q) = %d, %v; want %d, %v", tt.modelID, got, known, tt.want, tt.wantKnown)
			}
		})
	}
}

func TestWithCacheBreakpoints(t *testing.T) {
	marker := &CacheControl{Type: CacheControlTypeEphemeral}
	text := func(s string) Content { return Content{Type: MessageContentTypeText, Text: s} }
	markedDoc := Content{Type: MessageContentTypeDocument, Source: &Source{Type: SourceTypeText, Data: "doc"}, CacheControl: marker}
	thinking := Content{Type: MessageContentTypeThinking, Signature: "sig"}
	toolUse := Content{Type: MessageContentTypeToolUse, ID: "toolu_1", Name: "str_replace_based_edit_tool"}
	toolResult := Content{Type: "tool_result", ToolUseID: "toolu_1", Content: "file contents"}

	user := func(c ...Content) Message { return Message{Role: MessageRoleUser, Content: c} }
	assistant := func(c ...Content) Message { return Message{Role: MessageRoleAssistant, Content: c} }

	tests := []struct {
		name     string
		msgs     []Message
		markTail bool
		want     []string // "message:block" of each marked block
	}{
		{
			name:     "tail marker lands on the newest tool_result",
			msgs:     []Message{user(text("q")), assistant(thinking, toolUse), user(toolResult)},
			markTail: true,
			want:     []string{"2:0"},
		},
		{
			name: "no tail marker without markTail",
			msgs: []Message{user(text("q")), assistant(thinking, toolUse), user(toolResult)},
		},
		{
			name: "surplus explicit markers are dropped earliest first",
			msgs: []Message{user(markedDoc, markedDoc, markedDoc, markedDoc, markedDoc, text("q"))},
			want: []string{"0:1", "0:2", "0:3", "0:4"},
		},
		{
			name:     "tail marker takes one of the four slots",
			msgs:     []Message{user(markedDoc, markedDoc, markedDoc, markedDoc, text("q")), assistant(toolUse), user(toolResult)},
			markTail: true,
			want:     []string{"0:1", "0:2", "0:3", "2:0"},
		},
		{
			name:     "an already marked tail block counts once",
			msgs:     []Message{user(markedDoc, markedDoc, markedDoc, markedDoc)},
			markTail: true,
			want:     []string{"0:0", "0:1", "0:2", "0:3"},
		},
		{
			name:     "a thinking block is never marked",
			msgs:     []Message{user(text("q")), assistant(thinking)},
			markTail: true,
		},
		{
			name:     "no messages",
			markTail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, _ := json.Marshal(tt.msgs)

			got := withCacheBreakpoints(tt.msgs, tt.markTail)

			var marked []string
			for i, m := range got {
				for j, c := range m.Content {
					if c.CacheControl != nil {
						marked = append(marked, fmt.Sprintf("%d:%d", i, j))
					}
				}
			}
			if strings.Join(marked, " ") != strings.Join(tt.want, " ") {
				t.Errorf("marked blocks = %v, want %v", marked, tt.want)
			}
			if after, _ := json.Marshal(tt.msgs); string(after) != string(before) {
				t.Errorf("input messages were modified:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

func TestMessageStartUsageUnmarshal(t *testing.T) {
	event := `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","content":[],"model":"claude-opus-5-5","usage":{"input_tokens":12,"cache_creation_input_tokens":2048,"cache_read_input_tokens":18000,"output_tokens":1}}}`

	var got AnthropicClaudeMessagesResponse
	if err := json.Unmarshal([]byte(event), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Message == nil {
		t.Fatal("expected Message to be populated")
	}
	u := got.Message.Usage
	if u.InputTokens != 12 || u.CacheCreationInputTokens != 2048 || u.CacheReadInputTokens != 18000 {
		t.Errorf("usage = %+v, want input 12, cache_creation 2048, cache_read 18000", u)
	}
}
