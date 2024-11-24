package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePath(t *testing.T) {
	tool := NewEditTool()

	// Create temp test files/dirs
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(tmpFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		command     Command
		path        string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid file path for view",
			command:     View,
			path:        tmpFile,
			shouldError: false,
		},
		{
			name:        "Directory path for non-view command",
			command:     StrReplace,
			path:        tmpDir,
			shouldError: true,
			errorMsg:    "is a directory",
		},
		{
			name:        "Non-existent path for non-create command",
			command:     View,
			path:        filepath.Join(tmpDir, "nonexistent.txt"),
			shouldError: true,
			errorMsg:    "does not exist",
		},
		{
			name:        "Existing file path for create command",
			command:     Create,
			path:        tmpFile,
			shouldError: true,
			errorMsg:    "file already exists",
		},
		{
			name:        "Directory path for view command",
			command:     View,
			path:        tmpDir,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.validatePath(tt.command, tt.path)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestView(t *testing.T) {
	tool := NewEditTool()

	// Create temp test files/dirs
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4"
	err = os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		viewRange   []int
		shouldError bool
		errorMsg    string
		contains    string
	}{
		{
			name:      "View entire file",
			path:      tmpFile,
			viewRange: nil,
			contains:  "1\tLine 1\n     2\tLine 2\n     3\tLine 3\n     4\tLine 4\n",
		},
		{
			name:      "View specific range",
			path:      tmpFile,
			viewRange: []int{2, 3},
			contains:  "\n     2\tLine 2\n     3\tLine 3\n",
		},
		{
			name:        "Invalid range - start > end",
			path:        tmpFile,
			viewRange:   []int{3, 2},
			shouldError: true,
			errorMsg:    "invalid `view_range`",
		},
		{
			name:        "Invalid range - out of bounds",
			path:        tmpFile,
			viewRange:   []int{1, 10},
			shouldError: true,
			errorMsg:    "invalid `view_range`",
		},
		{
			name:     "View directory",
			path:     tmpDir,
			contains: "files and directories up to 2 levels deep",
		},
		{
			name:        "View directory with range",
			path:        tmpDir,
			viewRange:   []int{1, 2},
			shouldError: true,
			errorMsg:    "view_range` parameter is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := tool.view(tt.path, tt.viewRange)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, tt.contains)
			}
		})
	}
}
