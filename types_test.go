package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	nameClaude47Opus  = "Claude 4.7 Opus"
	nameClaude48Opus  = "Claude 4.8 Opus"
	nameClaudeFable5  = "Claude Fable 5"
	nameClaudeSonnet5 = "Claude Sonnet 5"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCitationsSupported(tt.modelID); got != tt.expected {
				t.Errorf("IsCitationsSupported(%q) = %v, want %v", tt.modelID, got, tt.expected)
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
