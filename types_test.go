package main

import (
	"testing"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVisionCapable(tt.modelID); got != tt.expected {
				t.Errorf("IsVisionCapable(%q) = %v, want %v", tt.modelID, got, tt.expected)
			}
		})
	}
}
