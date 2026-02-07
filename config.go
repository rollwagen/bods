package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/adrg/xdg"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/printer"
	"github.com/mattn/go-colorable"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

//go:embed bods.yaml
var bodsConfig []byte

// Global koanf instance;  "." as the key path delimiter
var k = koanf.New(".")

const (
	defaultMaxTokens      = 2048
	defaultThinkingTokens = 1024
	mininumThinkingTokens = 1024
)

type Config struct {
	Prefix               string
	ModelID              string // AnthropicModel
	SystemPrompt         string
	Assistant            string            // assistant messages
	Prompts              []Prompt          // prompts as defined in bods.yaml
	PromptTemplate       string            // name of prompt template (from config) to use
	UserPromptInputs     map[string]string // mapping of input variable to entered values e.g. {{.TASK}} => "Draft an email responding to a customer"
	MaxTokens            int               // max nr of tokens to generate before stopping
	Format               bool
	Metamode             bool
	Content              string // stdin input with rewritten and replaced variables; only used in metatprompt mode
	Pasteboard           bool
	ShowSettings         bool
	XMLTagContent        string
	CrossRegionInference bool
	Think                bool   // enables thinking (extended for 3.7-4.5, adaptive for Opus 4.6)
	BudgetTokens         int    // thinking budget tokens (3.7-4.5 only; deprecated for Opus 4.6)
	EnableTextEditor     bool   // enables text editor tool for Claude
	Effort               string // "max", "high", "medium", "low", or empty string

	ImagesFlagInput string // list of images e.g. file://image1.png,file://image2.jpeg
	ImageContent    []Content

	ToolCallJSONString string

	VariableInput    map[string]string // mapping of input variable to values
	VariableInputRaw string
}

// Prompt structure for for Anthropic Claude prompts
type Prompt struct {
	Name         string
	Description  string
	ModelID      string `koanf:"model_id"`
	Temperature  float64
	MaxTokens    int     `koanf:"max_tokens"`
	TopP         float64 `koanf:"top_p"`
	TopK         int     `koanf:"top_k"`
	System       string
	User         string
	Assistant    string
	Thinking     bool   `koanf:"thinking"`
	BudgetTokens int    `koanf:"budget_tokens"`
	TextEditor   bool   `koanf:"text_editor"`
	Effort       string `koanf:"effort"`
}

func newPrompt() Prompt {
	return Prompt{
		ModelID:      "anthropic.claude-v2:1",
		Temperature:  1,
		MaxTokens:    defaultMaxTokens,
		TopP:         0.999,
		Thinking:     false,
		BudgetTokens: defaultThinkingTokens,
		TextEditor:   false,
	}
}

func promptTemplateFieldValue[T string | int | float64 | bool](c *Config, field string) (T, bool) {
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

func configFilePath() string {
	logger.Println("config directories:", xdg.ConfigDirs)
	configFilePath, _ := xdg.ConfigFile("bods/bods.yaml")
	return configFilePath
}

func ensureConfig() (Config, error) {
	filePath := configFilePath()
	_, err := os.Stat(filePath)
	if err == nil {
		b, err := os.ReadFile(filePath)
		if err == nil {
			logger.Println("replacing standard embedded bods.yaml with file content from " + filePath)
			bodsConfig = b
		}
	}

	var c Config

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
		logger.Println("adding prompt from config:", name)
		c.Prompts = append(c.Prompts, p)
	}

	c.Format = true
	c.Metamode = false
	c.CrossRegionInference = true

	return c, nil
}

// printConfig prints the embedded bods.yaml config file to stdout
// yaml color output from https://github.com/goccy/go-yaml/blob/master/cmd/ycat/ycat.go
func printConfig(isTerminal bool) error {
	logger.Printf("printConfig: isTerminal = %v\n", isTerminal)
	if !isTerminal {
		_, err := fmt.Println(string(bodsConfig))
		if err != nil {
			return err
		}
		return nil
	}

	format := func(attr color.Attribute) string {
		const escape = "\x1b"
		return fmt.Sprintf("%s[%dm", escape, attr)
	}

	tokens := lexer.Tokenize(string(bodsConfig))
	var p printer.Printer
	p.LineNumber = false
	p.LineNumberFormat = func(num int) string {
		fn := color.New(color.Bold, color.FgHiWhite).SprintFunc()
		return fn(fmt.Sprintf("%2d | ", num))
	}
	p.Bool = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiMagenta),
			Suffix: format(color.Reset),
		}
	}
	p.Number = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiMagenta),
			Suffix: format(color.Reset),
		}
	}
	p.MapKey = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiCyan),
			Suffix: format(color.Reset),
		}
	}
	p.Anchor = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.Alias = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiYellow),
			Suffix: format(color.Reset),
		}
	}
	p.String = func() *printer.Property {
		return &printer.Property{
			Prefix: format(color.FgHiGreen),
			Suffix: format(color.Reset),
		}
	}

	fmt.Println("Embedded bods.yaml will be used and printed unless own config file exists.")
	fmt.Println("Config file path to replace embedded bods.yaml with own config: '" + configFilePath() + "'")

	writer := colorable.NewColorableStdout()
	_, err := writer.Write([]byte("\n" + p.PrintTokens(tokens) + "\n"))
	if err != nil {
		return err
	}

	return nil
}
