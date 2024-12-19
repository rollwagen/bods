package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func executeContextCommand(name string, arg ...string) (string, error) {
	logger.Printf("executeContextCommand(%s, %s)\n", name, arg)
	cmd := exec.Command(name, arg...)
	logger.Printf("executing context command '%s'\n", cmd.String())

	// executes the command and collects the output, returning its value
	out, err := cmd.Output()
	if err != nil {
		logger.Printf("'%s' failed with error: %v\n", cmd.String(), err)
		return "", err
	}
	// result := "\nOutput of command '" + cmd.String() + "' is:\n" + string(out) + "\n\n"
	result := "\n<context command='" + cmd.String() + "'>\n"
	result += string(out)
	result += "\n</context>\n\n"
	return result, nil
}

func executeContextCommands(commands []string) (string, error) {
	if len(commands) == 0 {
		logger.Println("executeContextCommands() - no commands provided")
		return "", fmt.Errorf("no commands provided")
	}

	var result string
	for _, command := range commands {
		// split command string by whitespaces
		commandParts := strings.Fields(command)
		out, err := executeContextCommand(commandParts[0], commandParts[1:]...)
		if err == nil {
			result += out
		}
	}
	response := "Here is useful information about the environment you are running in:\n\n"
	response += "\n<env>\n" + result + "\n</env>\n\n"
	return response, nil
}

// Here is useful information about the environment you are running in:
// <env>
// Working directory: /Users/rollwagen/Tmp/claude-code
// Is directory a git repo: No
// Platform: macos
// Today's date: 3/15/2025
// Model: us.anthropic.claude-3-7-sonnet-20250219-v1:0
// </env>
