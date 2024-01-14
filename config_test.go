package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureConfig(t *testing.T) {
	// Call function under test
	c, err := ensureConfig()

	// Assert results
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(c.Prompts), 1)
}
