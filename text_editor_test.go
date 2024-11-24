package main

import (
	"fmt"
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
	err = os.WriteFile(tmpFile, []byte("test content"), 0o644)
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
	err = os.WriteFile(tmpFile, []byte(content), 0o644)
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

func TestStrReplace(t *testing.T) {
	tool := NewEditTool()

	// Create temp test file
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")

	tests := []struct {
		name        string
		content     string
		oldStr      string
		newStr      string
		shouldError bool
		errorMsg    string
		expected    string
	}{
		{
			name:        "Replace unique string",
			content:     "Original content",
			oldStr:      "Original",
			newStr:      "New",
			shouldError: false,
			expected:    "New content",
		},
		{
			name:        "Non-existent string",
			content:     "Original content",
			oldStr:      "Nonexistent",
			newStr:      "New",
			shouldError: true,
			errorMsg:    "did not appear verbatim",
		},
		{
			name:        "Multiple occurrences",
			content:     "Test test test",
			oldStr:      "test",
			newStr:      "example",
			shouldError: true,
			errorMsg:    "Multiple occurrences",
		},
		{
			name:        "Empty old string",
			content:     "Some content",
			oldStr:      "",
			newStr:      "New",
			shouldError: true,
			errorMsg:    "old_str",
		},
		{
			name:        "Replace a string",
			content:     "Original content",
			oldStr:      "Original",
			newStr:      "New",
			shouldError: false,
			expected:    "New content",
			// errorMsg:    "old_str",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write initial content
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			assert.NoError(t, err)

			output, err := tool.strReplace(tmpFile, tt.oldStr, tt.newStr)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "has been edited")

				// Verify file content after replacement
				content, err := os.ReadFile(tmpFile)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, string(content))

				// Verify history was updated
				assert.Contains(t, tool.fileHistory[tmpFile], tt.content)
			}
		})
	}
}

func TestInsert(t *testing.T) {
	tool := NewEditTool()

	// Create temp test file
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")

	tests := []struct {
		name        string
		content     string
		insertLine  int
		newStr      string
		shouldError bool
		errorMsg    string
		expected    string
	}{
		{
			name:        "Insert at valid line number",
			content:     "Line 1\nLine 2\nLine 3",
			insertLine:  2,
			newStr:      "New Line",
			shouldError: false,
			expected:    "Line 1\nLine 2\nNew Line\nLine 3",
		},
		{
			name:        "Insert at beginning (line 0)",
			content:     "Line 1\nLine 2",
			insertLine:  0,
			newStr:      "New First Line",
			shouldError: false,
			expected:    "New First Line\nLine 1\nLine 2",
		},
		{
			name:        "Insert at end of file",
			content:     "Line 1\nLine 2",
			insertLine:  2,
			newStr:      "New Last Line",
			shouldError: false,
			expected:    "Line 1\nLine 2\nNew Last Line",
		},
		{
			name:        "Invalid line number - beyond EOF",
			content:     "Line 1\nLine 2",
			insertLine:  5,
			newStr:      "Invalid Line",
			shouldError: true,
			errorMsg:    "invalid `insert_line` parameter",
		},
		{
			name:        "Invalid line number - negative",
			content:     "Line 1\nLine 2",
			insertLine:  -1,
			newStr:      "Invalid Line",
			shouldError: true,
			errorMsg:    "invalid `insert_line` parameter",
		},
		{
			name:        "Standard insert",
			content:     "Line 1\nLine 2",
			insertLine:  1,
			newStr:      "New Line",
			shouldError: false,
			expected:    "Line 1\nNew Line\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write initial content
			err := os.WriteFile(tmpFile, []byte(tt.content), 0o644)
			assert.NoError(t, err)

			output, err := tool.insert(tmpFile, tt.insertLine, tt.newStr)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, "has been edited")

				// Verify file content after insertion
				content, err := os.ReadFile(tmpFile)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, string(content))

				// Verify history was updated
				assert.Contains(t, tool.fileHistory[tmpFile], tt.content)
			}
		})
	}
}

func TestUndoEdit(t *testing.T) {
	tool := NewEditTool()

	// Create temp test file
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")

	tests := []struct {
		name        string
		setupFunc   func()
		shouldError bool
		errorMsg    string
		contains    string
	}{
		{
			name: "Undo str_replace operation",
			setupFunc: func() {
				err := os.WriteFile(tmpFile, []byte("Original content"), 0o644)
				assert.NoError(t, err)
				_, err = tool.strReplace(tmpFile, "Original", "New")
				assert.NoError(t, err)
			},
			shouldError: false,
			contains:    "Original content",
		},
		{
			name: "Undo insert operation",
			setupFunc: func() {
				err := os.WriteFile(tmpFile, []byte("Line 1\nLine 2"), 0o644)
				assert.NoError(t, err)
				_, err = tool.insert(tmpFile, 1, "New Line")
				assert.NoError(t, err)
			},
			shouldError: false,
			contains:    "Line 1\nLine 2",
		},
		{
			name: "No edit history",
			setupFunc: func() {
				tool.fileHistory = make(map[string][]string)
				err := os.WriteFile(tmpFile, []byte("Test content"), 0o644)
				assert.NoError(t, err)
			},
			shouldError: true,
			errorMsg:    "No edit history found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()

			output, err := tool.undoEdit(tmpFile)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output, fmt.Sprintf("Last edit to %s undone successfully", tmpFile))
				// assert.Contains(t, output, tt.contains)

				// Verify file content after undo
				content, err := os.ReadFile(tmpFile)
				assert.NoError(t, err)
				assert.Contains(t, string(content), tt.contains)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	tool := NewEditTool()

	// Create temp test directory
	tmpDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Errorf("Error removing temp directory: %v", err)
		}
	}(tmpDir)

	tests := []struct {
		name        string
		path        string
		content     string
		setupFunc   func(string) error
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Write to new file",
			path:        filepath.Join(tmpDir, "new.txt"),
			content:     "Test content",
			shouldError: false,
		},
		{
			name:    "Write to existing file",
			path:    filepath.Join(tmpDir, "existing.txt"),
			content: "Updated content",
			setupFunc: func(path string) error {
				return os.WriteFile(path, []byte("Original content"), 0o644)
			},
			shouldError: false,
		},
		{
			name:    "Write to read-only directory",
			path:    filepath.Join(tmpDir, "readonly", "file.txt"),
			content: "Test content",
			setupFunc: func(path string) error {
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0o444); err != nil {
					return err
				}
				return nil
			},
			shouldError: true,
			errorMsg:    "while trying to write",
		},
		{
			name:        "Write to invalid path",
			path:        filepath.Join(tmpDir, string([]byte{0x0})),
			content:     "Test content",
			shouldError: true,
			errorMsg:    "while trying to write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				err := tt.setupFunc(tt.path)
				assert.NoError(t, err)
			}

			err := tool.writeFile(tt.path, tt.content)
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)

				// Verify file content
				content, err := os.ReadFile(tt.path)
				assert.NoError(t, err)
				assert.Equal(t, tt.content, string(content))
			}
		})
	}
}
