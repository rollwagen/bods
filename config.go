package main

import (
	_ "embed"
	"fmt"
	"log"
	"reflect"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

//go:embed bods.yaml
var bodsConfig []byte

// Global koanf instance;  "." as the key path delimiter
var k = koanf.New(".")

type Config struct {
	Prefix         string
	ModelID        string // AnthropicModel
	SystemPrompt   string
	Prompts        []Prompt // prompts as defined in bods.yaml
	PromptTemplate string   // name of prompt template (from config) to use
	MaxTokens      int      // max nr of tokens to generate before stopping
	Format         bool
	Pasteboard     bool
}

type Prompt struct {
	Name        string
	Description string
	ModelID     string `koanf:"model_id"`
	Temperature float64
	MaxTokens   int     `koanf:"max_tokens"`
	TopP        float64 `koanf:"top_p"`
	TopK        int     `koanf:"top_k"`
	System      string
	User        string
}

func newPrompt() Prompt {
	return Prompt{
		ModelID:     "anthropic.claude-v2:1",
		Temperature: 1,
		MaxTokens:   200,
		TopP:        0.999,
	}
}

func promptTemplateFieldValue[T string | int | float64](c *Config, field string) (T, bool) {
	var fieldValue T

	for _, p := range c.Prompts {
		if p.Name == c.PromptTemplate {
			v := reflect.ValueOf(p)
			f := v.FieldByName(field)
			if !f.IsValid() {
				logger.Println("PromptTemplateFieldValue: struct field does not exist - ", field)
				return fieldValue, false
			}
			fieldValue = f.Interface().(T)
			logger.Printf("PromptTemplateFieldValue: returning %s = %v\n", field, fieldValue)
			return fieldValue, true
		}
	}

	return fieldValue, false
}

func ensureConfig() (Config, error) {
	var c Config

	// if err := k.Load(file.Provider("./bods.yaml"), yaml.Parser()); err != nil {
	if err := k.Load(rawbytes.Provider(bodsConfig), yaml.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	for _, name := range k.MapKeys("prompts") {
		p := newPrompt() // var p Prompt
		p.Name = name
		err := k.Unmarshal(fmt.Sprintf("prompts.%s", name), &p)
		if err != nil {
			return Config{}, err
		}
		// logger.Println("adding prompt from config:", spew.Sprint(p))
		c.Prompts = append(c.Prompts, p)
	}

	c.Format = true

	return c, nil
}
