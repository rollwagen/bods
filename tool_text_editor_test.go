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
			name:         "Claude 4.7 Opus uses 20250728",
			model:        ClaudeV47Opus.String(),
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

// TestHandleTextEditorToolCallInputParsing exercises the tool end-to-end through
// HandleTextEditorToolCall, covering the input-parsing bugs where absent vs.
// zero-value params were conflated and view_range was dropped on a type mismatch.
func TestHandleTextEditorToolCallInputParsing(t *testing.T) {
	// GetTextEditorTool returns a process-global singleton; reset it so undo
	// history from other tests can't interfere.
	globalTextEditorTool = NewTextEditorTool()

	writeFile := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "sample.txt")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		return path
	}

	readFile := func(t *testing.T, path string) string {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read result file: %v", err)
		}
		return string(data)
	}

	call := func(t *testing.T, input map[string]any) *TextEditorToolResult {
		t.Helper()
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("failed to marshal tool input: %v", err)
		}
		return HandleTextEditorToolCall(raw)
	}

	t.Run("view_range returns only requested lines", func(t *testing.T) {
		path := writeFile(t, "line1\nline2\nline3\nline4\nline5")
		res := call(t, map[string]any{
			"command":    "view",
			"path":       path,
			"view_range": []int{2, 3},
		})
		if res.IsError {
			t.Fatalf("unexpected error: %s", res.Content)
		}
		if !strings.Contains(res.Content, "line2") || !strings.Contains(res.Content, "line3") {
			t.Errorf("expected lines 2-3 in output, got:\n%s", res.Content)
		}
		if strings.Contains(res.Content, "line1") || strings.Contains(res.Content, "line4") {
			t.Errorf("view_range not honored; full file returned:\n%s", res.Content)
		}
	})

	t.Run("insert_line 0 inserts at top", func(t *testing.T) {
		path := writeFile(t, "existing")
		res := call(t, map[string]any{
			"command":     "insert",
			"path":        path,
			"insert_line": 0,
			"new_str":     "header",
		})
		if res.IsError {
			t.Fatalf("unexpected error: %s", res.Content)
		}
		if got := readFile(t, path); got != "header\nexisting" {
			t.Errorf("expected header inserted at top, got %q", got)
		}
	})

	t.Run("str_replace with empty new_str deletes text", func(t *testing.T) {
		path := writeFile(t, "keep REMOVE keep")
		res := call(t, map[string]any{
			"command": "str_replace",
			"path":    path,
			"old_str": "REMOVE ",
			"new_str": "",
		})
		if res.IsError {
			t.Fatalf("unexpected error: %s", res.Content)
		}
		if got := readFile(t, path); got != "keep keep" {
			t.Errorf("expected matched text deleted, got %q", got)
		}
	})

	t.Run("create with empty file_text creates empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.txt")
		res := call(t, map[string]any{
			"command":   "create",
			"path":      path,
			"file_text": "",
		})
		if res.IsError {
			t.Fatalf("unexpected error: %s", res.Content)
		}
		if got := readFile(t, path); got != "" {
			t.Errorf("expected empty file, got %q", got)
		}
	})

	t.Run("insert accepts insert_text alias", func(t *testing.T) {
		path := writeFile(t, "existing")
		res := call(t, map[string]any{
			"command":     "insert",
			"path":        path,
			"insert_line": 0,
			"insert_text": "header",
		})
		if res.IsError {
			t.Fatalf("unexpected error: %s", res.Content)
		}
		if got := readFile(t, path); got != "header\nexisting" {
			t.Errorf("expected insert_text inserted at top, got %q", got)
		}
	})
}
