package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	snippetLinesMax = 4
)

type Command string

const (
	View       Command = "view"
	Create     Command = "create"
	StrReplace Command = "str_replace"
	Insert     Command = "insert"
	UndoEdit   Command = "undo_edit"
)

// EditTool represents the text editor tool for Anthropic's Claude Text Editor tool.
// Below is Anthropic's tool description to the Text Editor tool:
//
// Custom editing tool for viewing, creating and editing files
// * State is persistent across command calls and discussions with the user
// * If `path` is a file, `view` displays the result of applying `cat -n`. If `path` is a directory, `view` lists non-hidden files and directories up to 2 levels deep
// * The `create` command cannot be used if the specified `path` already exists as a file
// * If a `command` generates a long output, it will be truncated and marked with `<response clipped>`
// * The `undo_edit` command will revert the last edit made to the file at `path`
//
// Notes for using the `str_replace` command:
// * The `old_str` parameter should match EXACTLY one or more consecutive lines from the original file. Be mindful of whitespaces!
// * If the `old_str` parameter is not unique in the file, the replacement will not be performed. Make sure to include enough context in `old_str` to make it unique
// * The `new_str` parameter should contain the edited lines that should replace the `old_str`
type EditTool struct {
	fileHistory map[string][]string
}

func NewEditTool() *EditTool {
	return &EditTool{
		fileHistory: make(map[string][]string),
	}
}

func (t *EditTool) Call(command Command, path string, fileText string, viewRange []int, oldStr,
	newStr string, insertLine int,
) (string, error) {
	if t == nil {
		return "", errors.New("EditTool instance is nil")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	switch command {
	case View:
		err = t.validatePath(command, absPath)
		if err != nil {
			return "", err
		}
		return t.view(absPath, viewRange)
	case Create:
		if fileText == "" {
			return "", errors.New("parameter `file_text` is required for command: create")
		}
		err = t.validatePath(command, absPath)
		if err != nil {
			return "", err
		}
		err = t.writeFile(absPath, fileText)
		if err != nil {
			return "", err
		}
		t.fileHistory[absPath] = append(t.fileHistory[absPath], fileText)
		return fmt.Sprintf("file created successfully at: %s", absPath), nil
	case StrReplace:
		if oldStr == "" {
			return "", errors.New("parameter `old_str` is required for command: str_replace")
		}
		err = t.validatePath(command, absPath)
		if err != nil {
			return "", err
		}
		return t.strReplace(absPath, oldStr, newStr)
	case Insert:
		if insertLine < 0 {
			return "", errors.New("parameter `insert_line` is required for command: insert")
		}
		if newStr == "" {
			return "", errors.New("parameter `new_str` is required for command: insert")
		}
		err = t.validatePath(command, absPath)
		if err != nil {
			return "", err
		}
		return t.insert(absPath, insertLine, newStr)
	case UndoEdit:
		err = t.validatePath(command, absPath)
		if err != nil {
			return "", err
		}
		return t.undoEdit(absPath)
	default:
		return "", fmt.Errorf("unrecognized command %s", command)
	}
}

func (t *EditTool) validatePath(command Command, path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && command != Create {
			return fmt.Errorf("the path %s does not exist. Please provide a valid path", path)
		}
		if !os.IsNotExist(err) {
			return err // Return any other error
		}
	}

	if fileInfo != nil && fileInfo.IsDir() && command != View {
		return fmt.Errorf("the path %s is a directory and only the `view` command can be used on directories", path)
	}

	if fileInfo != nil && !fileInfo.IsDir() && command == Create {
		return fmt.Errorf("file already exists at: %s. Cannot overwrite files using command `create`", path)
	}

	return nil
}

func (t *EditTool) view(path string, viewRange []int) (string, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if fileInfo.IsDir() {
		if viewRange != nil {
			return "", errors.New("the `view_range` parameter is not allowed when `path` points to a directory")
		}

		cmd := fmt.Sprintf("find %s -maxdepth 2 -not -path '*/.*'", path)
		output, err := runCommand(cmd)
		if err != nil {
			return "", err
		}

		outputStr := fmt.Sprintf("Here's the files and directories up to 2 levels deep in %s, excluding hidden items:\n%s\n", path, output)
		return outputStr, nil
	}

	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	initLine := 1
	if viewRange != nil {
		if len(viewRange) != 2 {
			return "", errors.New("invalid `view_range`. It should be a list of two integers")
		}

		fileLines := strings.Split(string(fileContent), "\n")
		nLinesFile := len(fileLines)
		initLine = viewRange[0]
		finalLine := viewRange[1]

		if initLine < 1 || initLine > nLinesFile {
			return "", fmt.Errorf("invalid `view_range`: %v. Its first element `%d` should be within the range of lines of the file: %v", viewRange, initLine, []int{1, nLinesFile})
		}

		if finalLine > nLinesFile {
			return "", fmt.Errorf("invalid `view_range`: %v. Its second element `%d` should be smaller than the number of lines in the file: `%d`", viewRange, finalLine, nLinesFile)
		}

		if finalLine != -1 && finalLine < initLine {
			return "", fmt.Errorf("invalid `view_range`: %v. Its second element `%d` should be larger or equal than its first `%d`", viewRange, finalLine, initLine)
		}

		if finalLine == -1 {
			fileContent = []byte(strings.Join(fileLines[initLine-1:], "\n"))
		} else {
			fileContent = []byte(strings.Join(fileLines[initLine-1:finalLine], "\n"))
		}
	}
	output := t.makeOutput(string(fileContent), path, initLine)
	return output, nil
}

func (t *EditTool) strReplace(path, oldStr, newStr string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	fileContentStr := string(fileContent)
	occurrences := strings.Count(fileContentStr, oldStr)
	if occurrences == 0 {
		return "", fmt.Errorf("No replacement was performed, old_str `%s` did not appear verbatim in %s", oldStr, path)
	} else if occurrences > 1 {
		fileLines := strings.Split(fileContentStr, "\n")
		lines := []int{}
		for i, line := range fileLines {
			if strings.Contains(line, oldStr) {
				lines = append(lines, i+1)
			}
		}
		return "", fmt.Errorf("No replacement was performed. Multiple occurrences of old_str `%s` in lines %v. Please ensure it is unique", oldStr, lines)
	}

	newFileContent := strings.ReplaceAll(fileContentStr, oldStr, newStr)
	err = t.writeFile(path, newFileContent)
	if err != nil {
		return "", err
	}

	t.fileHistory[path] = append(t.fileHistory[path], fileContentStr)

	replacementLine := strings.Count(fileContentStr[:strings.Index(fileContentStr, oldStr)], "\n")
	startLine := max(0, replacementLine-snippetLinesMax)
	endLine := replacementLine + snippetLinesMax + strings.Count(newStr, "\n")
	snippet := strings.Join(strings.Split(newFileContent, "\n")[startLine:endLine+1], "\n")

	successMsg := fmt.Sprintf("The file %s has been edited. %sReview the changes and make sure they are as expected. Edit the file again if necessary.", path, t.makeOutput(snippet, fmt.Sprintf("a snippet of %s", path), startLine+1))
	return successMsg, nil
}

func (t *EditTool) insert(path string, insertLine int, newStr string) (string, error) {
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	fileLines := strings.Split(string(fileContent), "\n")
	nLinesFile := len(fileLines)

	if insertLine < 0 || insertLine > nLinesFile {
		return "", fmt.Errorf("invalid `insert_line` parameter: %d. It should be within the range of lines of the file: %v", insertLine, []int{0, nLinesFile})
	}

	newStrLines := strings.Split(newStr, "\n")
	newFileLines := append(append(fileLines[:insertLine], newStrLines...), fileLines[insertLine:]...)
	snippetLines := append(append(fileLines[max(0, insertLine-snippetLinesMax):insertLine], newStrLines...), fileLines[insertLine:insertLine+snippetLinesMax]...)

	newFileContent := strings.Join(newFileLines, "\n")
	snippet := strings.Join(snippetLines, "\n")

	err = t.writeFile(path, newFileContent)
	if err != nil {
		return "", err
	}
	t.fileHistory[path] = append(t.fileHistory[path], string(fileContent))

	successMsg := fmt.Sprintf("The file %s has been edited. %sReview the changes and make sure they are as expected (correct indentation, no duplicate lines, etc). "+
		"Edit the file again if necessary.",
		path, t.makeOutput(snippet, "a snippet of the edited file", max(1, insertLine-snippetLinesMax+1)),
	)
	return successMsg, nil
}

func (t *EditTool) undoEdit(path string) (string, error) {
	history, ok := t.fileHistory[path]
	if !ok || len(history) == 0 {
		return "", fmt.Errorf("No edit history found for %s", path)
	}

	oldText := history[len(history)-1]
	t.fileHistory[path] = history[:len(history)-1]
	err := t.writeFile(path, oldText)
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("Last edit to %s undone successfully. %s", path, t.makeOutput(oldText, path, 1))
	return output, nil
}

func (t *EditTool) writeFile(path, content string) error {
	err := os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		return fmt.Errorf("Ran into %v while trying to write to %s", err, path)
	}
	return nil
}

func (t *EditTool) makeOutput(fileContent, fileDescriptor string, initLine int) string {
	fileLines := strings.Split(fileContent, "\n")
	output := fmt.Sprintf("Here's the result of running `cat -n` on %s:\n", fileDescriptor)
	for i, line := range fileLines {
		// Use initLine + i to maintain original line numbers
		output += fmt.Sprintf("%6d\t%s\n", initLine+i, line)
	}
	return output
}
func runCommand(cmd string) (string, error) {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mainTextEditor() {
	// Usage example
	tool := NewEditTool()
	output, err := tool.Call(Create, "/path/to/file.txt", "Initial file content", nil, "", "", 0)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println(output)
}

// This Go code implements an `EditTool` struct with methods for viewing, creating, replacing strings, inserting text, and undoing edits in files. The `Call` method serves as the entry point and dispatches the appropriate method based on the provided command. The code handles various edge cases, such as validating file paths, checking for required parameters, and handling errors gracefully.

// Note that some helper functions like `execCommand` are left as placeholders and need to be implemented based on your specific requirements.
