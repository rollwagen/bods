package main

import (
	_ "embed"
	"fmt"
	"log"
	"reflect"

	"github.com/davecgh/go-spew/spew"
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
	Prompts        []Prompt
	PromptTemplate string // name of prompt template (from config) to use
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

func promptTemplateFieldValue[T string | int | float64](c *Config, field string) (T, bool) {
	var fieldValue T

	for _, p := range c.Prompts {
		if p.Name == c.PromptTemplate {
			v := reflect.ValueOf(p)
			f := v.FieldByName(field)
			if !f.IsValid() {
				logger.Println("PromptTamplateFieldValue: struct field does not exist - ", field)
				return fieldValue, false
			}
			fieldValue = f.Interface().(T)
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
		var p Prompt
		p.Name = name
		err := k.Unmarshal(fmt.Sprintf("prompts.%s", name), &p)
		if err != nil {
			return Config{}, err
		}
		logger.Println("adding prompt from config:", spew.Sprint(p))
		c.Prompts = append(c.Prompts, p)
	}

	return c, nil
}
