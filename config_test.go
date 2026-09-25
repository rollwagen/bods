package main

import (
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureConfig(t *testing.T) {
	// Call function under test
	c, err := ensureConfig()

	// Assert results
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(c.Prompts), 1)
}

func TestPromptWithoutModelID(t *testing.T) {
	// A template that leaves out model_id must stay unset, so startMessagesCmd falls through to
	// the default model instead of running a pre-filled one.
	kc := koanf.New(".")
	require.NoError(t, kc.Load(rawbytes.Provider([]byte("prompts:\n  mine:\n    user: hello\n")), yaml.Parser()))

	p := newPrompt()
	require.NoError(t, kc.Unmarshal("prompts.mine", &p))

	assert.Empty(t, p.ModelID)
	assert.Equal(t, "hello", p.User)
}
