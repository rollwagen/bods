package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//
// Tool API Constants and Types
//

// Constants for text editor API versions
const (
	TextEditor20241022 = "text_editor_20241022" // Claude 3.5 Sonnet
	TextEditor20250124 = "text_editor_20250124" // Claude 3.7 Sonnet
)

// TextEditorToolName is the fixed name required by Claude's API
const TextEditorToolName = "str_replace_editor"

// SnippetLines defines how many lines to show before/after edits
const SnippetLines = 4

// Command types supported by the editor
type EditorCommand string

const (
	ViewCommand       EditorCommand = "view"
	CreateCommand     EditorCommand = "create"
	StrReplaceCommand EditorCommand = "str_replace"
	InsertCommand     EditorCommand = "insert"
	UndoEditCommand   EditorCommand = "undo_edit"
)

//
// Tool Definitions
//

// TextEditorTool represents the text editor tool that enables Claude to edit files
type TextEditorTool struct {
	fileHistory map[string][]string // Path to history of file contents for undo
}

// TextEditorToolDefinition represents the tool definition to be sent to Claude's API
type TextEditorToolDefinition struct {
	Type string `json:"type"` // text_editor_20241022 or text_editor_20250124
	Name string `json:"name"` // Always "str_replace_editor"
}

// TextEditorToolCall represents a tool call from Claude
type TextEditorToolCall struct {
	Command    string         `json:"command"`
	Path       string         `json:"path"`
	FileText   string         `json:"file_text,omitempty"`
	ViewRange  []float64      `json:"view_range,omitempty"`
	OldStr     string         `json:"old_str,omitempty"`
	NewStr     string         `json:"new_str,omitempty"`
	InsertLine float64        `json:"insert_line,omitempty"`
	Parameters map[string]any `json:"-"` // For all other parameters
}

// TextEditorToolResult represents the result to send back to Claude
type TextEditorToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

//
// Tool Creation and Global Instance
//

// NewTextEditorTool creates a new text editor tool
func NewTextEditorTool() *TextEditorTool {
	return &TextEditorTool{
		fileHistory: make(map[string][]string),
	}
}

// NewTextEditorToolDefinition creates a new tool definition based on the Claude model
func NewTextEditorToolDefinition(model string) TextEditorToolDefinition {
	// Default to latest version
	toolType := TextEditor20250124

	// Use specific version based on model
	modelID := normalizeToModelID(model)
	if modelID == ClaudeV35Sonnet.String() || modelID == ClaudeV35SonnetV2.String() {
		toolType = TextEditor20241022
	}

	return TextEditorToolDefinition{
		Type: toolType,
		Name: TextEditorToolName,
	}
}

// Global instance of the text editor tool
var globalTextEditorTool *TextEditorTool

// GetTextEditorTool returns the global text editor tool instance
func GetTextEditorTool() *TextEditorTool {
	if globalTextEditorTool == nil {
		globalTextEditorTool = NewTextEditorTool()
	}
	return globalTextEditorTool
}

// -----------------------------------------------------------------------------
// API Integration
// -----------------------------------------------------------------------------

// HandleTextEditorToolCall processes a tool call from Claude and returns the result
func HandleTextEditorToolCall(ctx context.Context, toolCall json.RawMessage) (*TextEditorToolResult, error) {
	// Parse the tool call
	var call TextEditorToolCall
	if err := json.Unmarshal(toolCall, &call); err != nil {
		return &TextEditorToolResult{
			Content: "Error parsing tool call: " + err.Error(),
			IsError: true,
		}, nil
	}

	// Create a map of parameters for the tool
	params := make(map[string]any)
	if call.FileText != "" {
		params["file_text"] = call.FileText
	}
	if call.ViewRange != nil {
		params["view_range"] = call.ViewRange
	}
	if call.OldStr != "" {
		params["old_str"] = call.OldStr
	}
	if call.NewStr != "" {
		params["new_str"] = call.NewStr
	}
	if call.InsertLine != 0 {
		params["insert_line"] = call.InsertLine
	}

	// Create or get the text editor tool
	tool := GetTextEditorTool()

	// Execute the command
	result, err := tool.ExecuteCommand(ctx, EditorCommand(call.Command), call.Path, params)
	if err != nil {
		return &TextEditorToolResult{
			Content: "Error: " + err.Error(),
			IsError: true,
		}, nil
	}

	return &TextEditorToolResult{
		Content: result,
		IsError: false,
	}, nil
}

// -----------------------------------------------------------------------------
// Command Execution
// -----------------------------------------------------------------------------

// ExecuteCommand handles incoming editor commands
func (t *TextEditorTool) ExecuteCommand(ctx context.Context, command EditorCommand, path string, params map[string]any) (string, error) {
	// Validate path
	if err := t.validatePath(command, path); err != nil {
		return "", err
	}

	switch command {
	case ViewCommand:
		return t.view(path, params)
	case CreateCommand:
		fileText, ok := params["file_text"].(string)
		if !ok {
			return "", errors.New("parameter 'file_text' is required for command: create")
		}
		return t.create(path, fileText)
	case StrReplaceCommand:
		oldStr, ok := params["old_str"].(string)
		if !ok {
			return "", errors.New("parameter 'old_str' is required for command: str_replace")
		}
		newStr, ok := params["new_str"].(string)
		if !ok {
			newStr = ""
		}
		return t.strReplace(path, oldStr, newStr)
	case InsertCommand:
		insertLine, ok := params["insert_line"].(float64)
		if !ok {
			return "", errors.New("parameter 'insert_line' is required for command: insert")
		}
		newStr, ok := params["new_str"].(string)
		if !ok {
			return "", errors.New("parameter 'new_str' is required for command: insert")
		}
		return t.insert(path, int(insertLine), newStr)
	case UndoEditCommand:
		return t.undoEdit(path)
	default:
		return "", fmt.Errorf("unrecognized command: %s", command)
	}
}

// validatePath checks that the path/command combination is valid
func (t *TextEditorTool) validatePath(command EditorCommand, path string) error {
	// Check if it's an absolute path
	if !filepath.IsAbs(path) {
		suggestedPath := filepath.Join("", path)
		return fmt.Errorf("the path %s is not an absolute path, it should start with '/'. Maybe you meant %s?", path, suggestedPath)
	}

	// Check if path exists (except for create command)
	if _, err := os.Stat(path); os.IsNotExist(err) && command != CreateCommand {
		return fmt.Errorf("the path %s does not exist. Please provide a valid path", path)
	}

	// Check if file already exists for create command
	if _, err := os.Stat(path); err == nil && command == CreateCommand {
		return fmt.Errorf("file already exists at: %s. Cannot overwrite files using command 'create'", path)
	}

	// Check if the path points to a directory
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() && command != ViewCommand {
		return fmt.Errorf("the path %s is a directory and only the 'view' command can be used on directories", path)
	}

	return nil
}

//
// Command Implementations
//

// view implements the view command
func (t *TextEditorTool) view(path string, params map[string]any) (string, error) {
	// Handle directory view
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		if _, ok := params["view_range"]; ok {
			return "", errors.New("the 'view_range' parameter is not allowed when 'path' points to a directory")
		}

		// List directory contents
		files, err := os.ReadDir(path)
		if err != nil {
			return "", fmt.Errorf("error reading directory %s: %v", path, err)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Directory contents of %s:\n", path))
		for _, file := range files {
			filePath := filepath.Join(path, file.Name())
			fileType := "file"
			if file.IsDir() {
				fileType = "dir"
			}
			sb.WriteString(fmt.Sprintf("%s (%s)\n", filePath, fileType))
		}
		return sb.String(), nil
	}

	// Read file content
	content, err := t.readFile(path)
	if err != nil {
		return "", err
	}

	// Handle view_range if provided
	lines := strings.Split(content, "\n")
	startLine := 1
	endLine := len(lines)

	if viewRange, ok := params["view_range"].([]any); ok {
		if len(viewRange) != 2 {
			return "", errors.New("invalid 'view_range'. It should be a list of two integers")
		}

		// Extract and validate start line
		if startFloat, ok := viewRange[0].(float64); ok {
			startLine = int(startFloat)
		} else {
			return "", errors.New("invalid 'view_range'. The first element should be a number")
		}

		if startLine < 1 || startLine > len(lines) {
			return "", fmt.Errorf("invalid 'view_range': %v. Its first element '%d' should be within the range of lines of the file: [1, %d]", viewRange, startLine, len(lines))
		}

		// Extract and validate end line
		if endFloat, ok := viewRange[1].(float64); ok {
			endLine = int(endFloat)
			if endLine != -1 {
				if endLine > len(lines) {
					return "", fmt.Errorf("invalid 'view_range': %v. Its second element '%d' should be smaller than the number of lines in the file: '%d'", viewRange, endLine, len(lines))
				}
				if endLine < startLine {
					return "", fmt.Errorf("invalid 'view_range': %v. Its second element '%d' should be larger or equal than its first '%d'", viewRange, endLine, startLine)
				}
			} else {
				endLine = len(lines)
			}
		} else {
			return "", errors.New("invalid 'view_range'. The second element should be a number")
		}

		// Extract specified lines
		if endLine == -1 {
			lines = lines[startLine-1:]
		} else {
			lines = lines[startLine-1 : endLine]
		}
		content = strings.Join(lines, "\n")
	}

	return t.makeOutput(content, path, startLine), nil
}

// create implements the create command
func (t *TextEditorTool) create(path string, fileText string) (string, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Write the file
	err := t.writeFile(path, fileText)
	if err != nil {
		return "", err
	}

	// Store content in history
	t.fileHistory[path] = append(t.fileHistory[path], fileText)

	return fmt.Sprintf("File created successfully at: %s", path), nil
}

// strReplace implements the str_replace command
func (t *TextEditorTool) strReplace(path string, oldStr string, newStr string) (string, error) {
	// Read the file content
	fileContent, err := t.readFile(path)
	if err != nil {
		return "", err
	}

	// Check if oldStr is unique in the file
	occurrences := strings.Count(fileContent, oldStr)
	if occurrences == 0 {
		return "", fmt.Errorf("no replacement was performed, old_str '%s' did not appear verbatim in %s", oldStr, path)
	} else if occurrences > 1 {
		// Find line numbers for better error messages
		lines := strings.Split(fileContent, "\n")
		lineNumbers := []int{}
		for i, line := range lines {
			if strings.Contains(line, oldStr) {
				lineNumbers = append(lineNumbers, i+1)
			}
		}
		return "", fmt.Errorf("no replacement was performed. Multiple occurrences of old_str '%s' in lines %v. Please ensure it is unique", oldStr, lineNumbers)
	}

	// Save the current content for undo
	t.fileHistory[path] = append(t.fileHistory[path], fileContent)

	// Replace oldStr with newStr
	newFileContent := strings.Replace(fileContent, oldStr, newStr, 1)

	// Write the new content to the file
	err = t.writeFile(path, newFileContent)
	if err != nil {
		return "", err
	}

	// Create a snippet of the edited section
	replacementLine := strings.Count(strings.Split(fileContent, oldStr)[0], "\n")
	startLine := max(0, replacementLine-SnippetLines)
	endLine := replacementLine + SnippetLines + strings.Count(newStr, "\n")

	snippetLines := strings.Split(newFileContent, "\n")
	if endLine+1 > len(snippetLines) {
		endLine = len(snippetLines) - 1
	}
	snippet := strings.Join(snippetLines[startLine:endLine+1], "\n")

	successMsg := fmt.Sprintf("The file %s has been edited.\n", path)
	successMsg += t.makeOutput(snippet, fmt.Sprintf("a snippet of %s", path), startLine+1)
	successMsg += "Review the changes and make sure they are as expected. Edit the file again if necessary."

	return successMsg, nil
}

// insert implements the insert command
func (t *TextEditorTool) insert(path string, insertLine int, newStr string) (string, error) {
	// Read the file content
	fileContent, err := t.readFile(path)
	if err != nil {
		return "", err
	}

	// Split into lines
	lines := strings.Split(fileContent, "\n")

	// Validate insert line
	if insertLine < 0 || insertLine > len(lines) {
		return "", fmt.Errorf("invalid 'insert_line' parameter: %d. It should be within the range of lines of the file: [0, %d]", insertLine, len(lines))
	}

	// Save the current content for undo
	t.fileHistory[path] = append(t.fileHistory[path], fileContent)

	// Insert new string at the specified location
	newLines := strings.Split(newStr, "\n")

	// Combine before, insert and after
	resultLines := make([]string, 0, len(lines)+len(newLines))
	resultLines = append(resultLines, lines[:insertLine]...)
	resultLines = append(resultLines, newLines...)
	resultLines = append(resultLines, lines[insertLine:]...)

	newFileContent := strings.Join(resultLines, "\n")

	// Write the new content to the file
	err = t.writeFile(path, newFileContent)
	if err != nil {
		return "", err
	}

	// Create snippet for output
	snippetStartLine := max(0, insertLine-SnippetLines)
	snippetEndLine := min(len(resultLines), insertLine+len(newLines)+SnippetLines)

	snippetLines := resultLines[snippetStartLine:snippetEndLine]
	snippet := strings.Join(snippetLines, "\n")

	successMsg := fmt.Sprintf("The file %s has been edited.\n", path)
	successMsg += t.makeOutput(snippet, "a snippet of the edited file", snippetStartLine+1)
	successMsg += "Review the changes and make sure they are as expected (correct indentation, no duplicate lines, etc). Edit the file again if necessary."

	return successMsg, nil
}

// undoEdit implements the undo_edit command
func (t *TextEditorTool) undoEdit(path string) (string, error) {
	if len(t.fileHistory[path]) == 0 {
		return "", fmt.Errorf("no edit history found for %s", path)
	}

	// Get the last saved content
	historyLen := len(t.fileHistory[path])
	oldText := t.fileHistory[path][historyLen-1]

	// Remove the last item from history
	t.fileHistory[path] = t.fileHistory[path][:historyLen-1]

	// Write the old content back to the file
	err := t.writeFile(path, oldText)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Last edit to %s undone successfully.\n%s", path, t.makeOutput(oldText, path, 1)), nil
}

//
// Helper Functions
//

// readFile reads a file and returns its content
func (t *TextEditorTool) readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ran into %v while trying to read %s", err, path)
	}
	return string(data), nil
}

// writeFile writes content to a file
func (t *TextEditorTool) writeFile(path string, content string) error {
	err := os.WriteFile(path, []byte(content), 0o600) // before: 0o644) but G306 Expect 0600 or less
	if err != nil {
		return fmt.Errorf("ran into %v while trying to write to %s", err, path)
	}
	return nil
}

// makeOutput formats the output with line numbers
func (t *TextEditorTool) makeOutput(content string, fileDescriptor string, initialLine int) string {
	lines := strings.Split(content, "\n")
	numberedLines := make([]string, len(lines))

	for i, line := range lines {
		numberedLines[i] = fmt.Sprintf("%6d\t%s", initialLine+i, line)
	}

	numberedContent := strings.Join(numberedLines, "\n")
	return fmt.Sprintf("Here's the result of running `cat -n` on %s:\n%s\n", fileDescriptor, numberedContent)
}
