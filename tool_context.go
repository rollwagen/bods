package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ToolDirectoryContext returns a directory structure context string
// of the current working directory for use with Claude.
func ToolWorkingDirectoryContext() string {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("Error getting working directory: %v\n", err)
	}

	contextContent := fmt.Sprintf(`

As you answer the user's questions, you can use the following context:

<context name="directoryStructure">

	Below is a snapshot of this project's file structure at the start of the conversation.
	This snapshot will NOT update during the conversation.
	It skips over .gitignore patterns.

	%s

</context>

`, generateDirectoryStructure(workDir))

	return contextContent
}

// generateDirectoryStructure creates a formatted directory structure
// representation starting from the specified root directory.
func generateDirectoryStructure(rootDir string) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("- %s/\n", rootDir))

	// Get gitignore patterns if available
	ignorePatterns := getGitIgnorePatterns(rootDir)

	// Read the directory entries
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return fmt.Sprintf("  Error reading directory: %v\n", err)
	}

	// Sort entries alphabetically
	sortedEntries := make([]os.DirEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Name() < sortedEntries[j].Name()
	})

	// Process each entry
	for _, entry := range sortedEntries {
		name := entry.Name()

		// Skip items that match gitignore patterns
		if shouldIgnore(name, ignorePatterns) {
			continue
		}

		// Skip .git directory
		if name == ".git" {
			continue
		}

		// Format the entry
		entryPath := filepath.Join(rootDir, name)
		if entry.IsDir() {
			processDirectory(entryPath, "  ", ignorePatterns, &result)
		} else {
			result.WriteString(fmt.Sprintf("  - %s\n", name))
		}
	}

	return result.String()
}

// processDirectory recursively processes a directory and adds its contents
// to the result string with proper indentation.
func processDirectory(dirPath string, indent string, ignorePatterns []string, result *strings.Builder) {
	// Add directory name with trailing slash
	dirName := filepath.Base(dirPath)
	result.WriteString(fmt.Sprintf("%s- %s/\n", indent, dirName))

	// Read the directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		result.WriteString(fmt.Sprintf("%s  Error reading directory: %v\n", indent, err))
		return
	}

	// Sort entries alphabetically
	sortedEntries := make([]os.DirEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Name() < sortedEntries[j].Name()
	})

	// Process each entry with increased indentation
	nextIndent := indent + "  "
	for _, entry := range sortedEntries {
		name := entry.Name()

		// Skip items that match gitignore patterns
		if shouldIgnore(name, ignorePatterns) {
			continue
		}

		// Format the entry
		entryPath := filepath.Join(dirPath, name)
		if entry.IsDir() {
			processDirectory(entryPath, nextIndent, ignorePatterns, result)
		} else {
			result.WriteString(fmt.Sprintf("%s- %s\n", nextIndent, name))
		}
	}
}

// getGitIgnorePatterns reads .gitignore file and returns its patterns.
func getGitIgnorePatterns(rootDir string) []string {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return []string{}
	}

	lines := strings.Split(string(content), "\n")
	var patterns []string
	for _, line := range lines {
		// Skip empty lines and comments
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			patterns = append(patterns, trimmedLine)
		}
	}

	// Add custom patterns from CLAUDE.md if it exists
	claudeMdPath := filepath.Join(rootDir, "CLAUDE.md")
	claudeContent, err := os.ReadFile(claudeMdPath)
	if err == nil {
		claudeLines := strings.Split(string(claudeContent), "\n")
		for _, line := range claudeLines {
			if strings.Contains(line, "Ignore all files") {
				parts := strings.Split(line, "Ignore all files")
				if len(parts) > 1 {
					patterns = append(patterns, strings.TrimSpace(parts[1]))
				}
			}
		}
	}

	return patterns
}

// shouldIgnore checks if a file or directory should be ignored
// based on the provided patterns.
func shouldIgnore(name string, patterns []string) bool {
	// Check each pattern
	for _, pattern := range patterns {
		// Handle negated patterns
		if strings.HasPrefix(pattern, "!") {
			continue
		}

		// Handle directory-specific patterns
		if strings.HasSuffix(pattern, "/") {
			if name == strings.TrimSuffix(pattern, "/") {
				return true
			}
			continue
		}

		// Handle file extension patterns
		if strings.HasPrefix(pattern, "*.") {
			extension := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, extension) {
				return true
			}
			continue
		}

		// Handle exact matches
		if pattern == name {
			return true
		}

		// Handle wildcard patterns
		if strings.Contains(pattern, "*") {
			// Simple implementation - could be enhanced with more robust pattern matching
			beforeStar := strings.Split(pattern, "*")[0]
			afterStar := strings.Split(pattern, "*")[1]
			if strings.HasPrefix(name, beforeStar) && strings.HasSuffix(name, afterStar) {
				return true
			}
		}
	}

	return false
}
