package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
			name:         "Claude 4.5 Opus uses 20250728",
			model:        ClaudeV45Opus.String(),
			expectedType: TextEditor20250728,
			expectedName: TextEditorToolNameNew,
		},
		{
			name:         "Claude 4.6 Opus uses 20250728",
			model:        ClaudeV46Opus.String(),
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

func TestHandleTextEditorToolCall_InsertText(t *testing.T) {
	// Reset global instance for clean test
	globalTextEditorTool = nil

	// Create a temporary file to insert into
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("line 1\nline 2\nline 3\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Simulate Claude 4+ insert command JSON using insert_text (not new_str)
	toolCall := map[string]any{
		"command":     "insert",
		"path":        testFile,
		"insert_line": 1,
		"insert_text": "inserted line",
	}
	rawJSON, err := json.Marshal(toolCall)
	if err != nil {
		t.Fatalf("failed to marshal tool call: %v", err)
	}

	result := HandleTextEditorToolCall(rawJSON)
	if result.IsError {
		t.Fatalf("expected success, got error: %s", result.Content)
	}

	// Verify the file was modified correctly
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read test file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[1] != "inserted line" {
		t.Errorf("expected 'inserted line' at line 2, got file content:\n%s", string(content))
	}
}
