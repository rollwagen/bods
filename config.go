package main

type Config struct {
	Prefix  string
	ModelID string // AnthropicModel
}

func ensureConfig() (Config, error) { //nolint:unparam
	c := Config{
		Prefix: "",
	}
	return c, nil
}

type AnthropicModel int

const (
	Undefined AnthropicModel = iota
	ClaudeV1
	ClaudeV2
	ClaudeV21
)

func (m AnthropicModel) String() string {
	switch m {
	case ClaudeV1:
		return "anthropic.claude-v1"
	case ClaudeV2:
		return "anthropic.claude-v2"
	case ClaudeV21:
		return "anthropic.claude-v2:1"
	}
	return "unknown"
}

var AnthophicModelsIDs = []string{
	ClaudeV1.String(),
	ClaudeV2.String(),
	ClaudeV21.String(),
}
