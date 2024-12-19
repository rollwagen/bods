package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func executeContextCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	logger.Printf("executing context command '%s'\n", cmd.String())

	// executes the command and collects the output, returning its value
	out, err := cmd.Output()
	if err != nil {
		logger.Println("ERROR", err)
		fmt.Println("ERROR", err)
		return "", err
	}
	result := "\nOutput of command '" + cmd.String() + "' is:\n" + string(out) + "\n\n"
	return result, nil
}

func executeContextCommands(commands []string) (string, error) {
	// if commands == nil || len(commands) == 0 {
	if len(commands) == 0 {
		logger.Println("executeContextCommands() - no commands provided")
		return "", fmt.Errorf("no commands provided")
	}

	var result string
	for _, command := range commands {
		// split command string by whitespaces
		commandParts := strings.Fields(command)
		out, err := executeContextCommand(commandParts[0], commandParts[1:]...)
		if err != nil {
			return "", err
		}
		result += out
	}
	response := "\n<context>\n" + result + "\n</context>\n\n"
	return response, nil
}
