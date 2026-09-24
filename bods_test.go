package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRefusalError(t *testing.T) {
	tests := []struct {
		name     string
		category string
		contains []string
	}{
		{
			name:     "no category",
			category: "",
			contains: []string{"declined the request", "rephrasing"},
		},
		{
			name:     "safeguard category is named",
			category: "bio",
			contains: []string{"declined the request", "bio"},
		},
		{
			name:     "reasoning_extraction explains the cause",
			category: "reasoning_extraction",
			contains: []string{"reasoning_extraction", "internal reasoning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := refusalError(tt.category)
			if !errors.Is(err, errRefused) {
				t.Errorf("refusalError(%q) does not wrap errRefused: %v", tt.category, err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("refusalError(%q) = %q, want it to contain %q", tt.category, err, want)
				}
			}
		})
	}
}
