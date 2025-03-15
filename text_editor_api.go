package main

import (
	"context"
	"encoding/json"
)

// Constants for text editor API versions
const (
	TextEditor20241022 = "text_editor_20241022" // Claude 3.5 Sonnet
	TextEditor20250124 = "text_editor_20250124" // Claude 3.7 Sonnet
)

// TextEditorToolName is the fixed name required by Claude's API
const TextEditorToolName = "str_replace_editor"

// TextEditorToolDefinition represents the tool definition to be sent to Claude's API
type TextEditorToolDefinition struct {
	Type string `json:"type"` // text_editor_20241022 or text_editor_20250124
	Name string `json:"name"` // Always "str_replace_editor"
}

// TextEditorToolCall represents a tool call from Claude
type TextEditorToolCall struct {
	Command    string                 `json:"command"`
	Path       string                 `json:"path"`
	FileText   string                 `json:"file_text,omitempty"`
	ViewRange  []float64              `json:"view_range,omitempty"`
	OldStr     string                 `json:"old_str,omitempty"`
	NewStr     string                 `json:"new_str,omitempty"`
	InsertLine float64                `json:"insert_line,omitempty"`
	Parameters map[string]interface{} `json:"-"` // For all other parameters
}

// TextEditorToolResult represents the result to send back to Claude
type TextEditorToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
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
	params := make(map[string]interface{})
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

// Global instance of the text editor tool
var globalTextEditorTool *TextEditorTool

// GetTextEditorTool returns the global text editor tool instance
func GetTextEditorTool() *TextEditorTool {
	if globalTextEditorTool == nil {
		globalTextEditorTool = NewTextEditorTool()
	}
	return globalTextEditorTool
}