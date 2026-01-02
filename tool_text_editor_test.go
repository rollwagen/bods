package main

import (
	"testing"
)

func TestNewTextEditorToolDefinition(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		expectedType string
		expectedName string
	}{
		{
			name:         "Claude 3.5 Sonnet V1 uses 20241022",
			model:        ClaudeV35Sonnet.String(),
			expectedType: TextEditor20241022,
			expectedName: TextEditorToolNameLegacy,
		},
		{
			name:         "Claude 3.5 Sonnet V2 uses 20241022",
			model:        ClaudeV35SonnetV2.String(),
			expectedType: TextEditor20241022,
			expectedName: TextEditorToolNameLegacy,
		},
		{
			name:         "Claude 3.7 Sonnet uses 20250124",
			model:        ClaudeV37Sonnet.String(),
			expectedType: TextEditor20250124,
			expectedName: TextEditorToolNameLegacy,
		},
		{
			name:         "Claude 4 Sonnet uses 20250728",
			model:        ClaudeV4Sonnet.String(),
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Claude 4 Opus uses 20250728",
			model:        ClaudeV4Opus.String(),
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Claude 4.5 Sonnet uses 20250728",
			model:        ClaudeV45Sonnet.String(),
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Claude 4.5 Haiku uses 20250728",
			model:        ClaudeV45Haiku.String(),
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Regional inference profile for Claude 4.5 Sonnet uses 20250728",
			model:        "eu.anthropic.claude-sonnet-4-5-20250929-v1:0",
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Global inference profile for Claude 4.5 Sonnet uses 20250728",
			model:        "global.anthropic.claude-sonnet-4-5-20250929-v1:0",
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewTextEditorToolDefinition(tt.model)

			if result.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s for model %s", tt.expectedType, result.Type, tt.model)
			}

			if result.Name != tt.expectedName {
				t.Errorf("Expected name %s, got %s for model %s", tt.expectedName, result.Name, tt.model)
			}
		})
	}
}
