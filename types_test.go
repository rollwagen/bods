package main

import (
	"testing"
)

const (
	nameClaude47Opus = "Claude 4.7 Opus"
	nameClaude48Opus = "Claude 4.8 Opus"
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
