package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteContextCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		args     []string
		expected string
		wantErr  bool
	}{
		{
			name:     "Echo command success",
			command:  "echo",
			args:     []string{"hello"},
			expected: "\n<context>\nhello\n</context>\n",
			wantErr:  false,
		},
		{
			name:     "Multiple args command",
			command:  "echo",
			args:     []string{"hello", "world"},
			expected: "\n<context>\nhello world\n</context>\n",
			wantErr:  false,
		},
		{
			name:     "Invalid command",
			command:  "invalidcommand",
			args:     []string{},
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty command",
			command:  "",
			args:     []string{},
			expected: "",
			wantErr:  true,
		},
		{
			name:    "Tree comamnd",
			command: "tree",
			args:    []string{"-L", "3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeContextCommand(tt.command, tt.args...)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				if tt.expected != "" {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}
