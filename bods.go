package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// _ "x/image/webp"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	// "github.com/rollwagen/bods/pasteboard"
)

var (
	errContextCanceled     = errors.New("context was canceled")
	errEmptyResponseStream = errors.New("response stream was empty (nil)")
)

type state int

var (
	messages          = []Message{{Role: MessageRoleUser}}
	paramsMessagesAPI = NewAnthropicClaudeMessagesInferenceParameters()

	bedrockRuntimeClient *bedrockruntime.Client
)

const (
	startState state = iota
	doneState
	requestState
	responseState
	errorState
)

// bodsError is a wrapper around error adding additional context.
type bodsError struct {
	err    error
	reason string
}

func (m bodsError) Error() string {
	return m.err.Error()
}

// Bods is the Bubble Tea model that manages reading stdin and querying bedrock
type Bods struct {
	Output        string
	Input         string
	Styles        styles
	Error         *bodsError
	state         state
	glam          *glamour.TermRenderer
	glamOutput    string
	cancelRequest context.CancelFunc
	context       *context.Context

	Config *Config
}

func (b *Bods) Init() tea.Cmd {
	b.Update(nil)
	return nil
}

func (b *Bods) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logger.Println("Update() ...") // logger.Printf("Update() msg.content='%v'\n", msg)
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case promptInput:
		if msg.content != "" {
			b.Input = msg.content
		}
		if msg.content == "" && b.Config.Prefix == "" {
			return b, b.quit
		}
		b.state = requestState
		cmds = append(cmds, b.startMessagesCmd(msg.content))

	case completionOutput:
		logger.Printf("completionOutput content=%s\n", msg.content)
		if msg.content != "" {
			b.Output += msg.content
			if isOutputTerminal() {
				if b.Config.Format {
					b.glamOutput, _ = b.glam.Render(b.Output)
				} else {
					b.glamOutput = b.Output
				}
				b.state = responseState
			}
		}
		if msg.stream == nil {
			if b.Config.XMLTagContent != "" {
				// if b.Config.Metamode && b.Config.PromptTemplate == "metaprompt" {
				content := extractXMLTagContent(b.Output, b.Config.XMLTagContent)
				file, _ := os.Create(b.Config.XMLTagContent + ".txt")
				// _, _ = file.WriteString(b.Config.XMLTagContent)
				_, _ = file.WriteString(content)
				_ = file.Close()
			}
			b.state = doneState
			return b, b.quit
		}
		cmds = append(cmds, b.receiveStreamingMessagesCmd(msg))

	case bodsError:
		b.Error = &msg
		b.state = errorState
		return b, b.quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			b.state = doneState
			return b, b.quit
		}
	}

	if b.state == startState {
		logger.Println("current state is 'startState', appending 'readStdinCmd'")
		cmds = append(cmds, readStdinCmd)
	}

	return b, tea.Batch(cmds...)
}

func (b *Bods) View() string {
	switch b.state {
	case responseState:
		if isOutputTerminal() {
			return b.glamOutput
		}
	case errorState:
		return ""
	}
	return ""
}

func (b *Bods) quit() tea.Msg {
	if b.cancelRequest != nil {
		b.cancelRequest()
	}
	return tea.Quit()
}

func (b *Bods) startMessagesCmd(content string) tea.Cmd {
	// content is piped input e.g. echo "content" | bods
	logger.Printf("startMessagesCmd: len(content)=%d\n", len(content))

	awsConfig, err := sdkconfig.LoadDefaultConfig(*b.context)
	if err != nil {
		msg := fmt.Sprintf("LoadDefaultConfig(): failed to load SDK configuration, %v", err)
		log.Fatalf("%s", msg)
	}
	bedrockRuntimeClient = bedrockruntime.NewFromConfig(awsConfig)
	bedrockClient := bedrock.NewFromConfig(awsConfig)

	return func() tea.Msg {
		// ORIG LOCATION paramsMessagesAPI := NewAnthropicClaudeMessagesInferenceParameters()

		const defaultMarkdownFormatText = " Format the response as markdown without enclosing backticks."

		// use model as specified in prompt template, unless overridden with '--model' flag
		promptTemplateModelID, _ := promptTemplateFieldValue[string](b.Config, "ModelID")
		if b.Config.ModelID == "" && promptTemplateModelID != "" {
			b.Config.ModelID = promptTemplateModelID
		}
		if b.Config.ModelID == "" { // initialize to default if no modelID given at all
			// b.Config.ModelID = ClaudeV35SonnetV2.String()
			// b.Config.ModelID = ClaudeV37Sonnet.String()
			// b.Config.ModelID = ClaudeV4Sonnet.String()
			b.Config.ModelID = ClaudeV45Sonnet.String()
		}
		logger.Println("config.ModelID set to: ", b.Config.ModelID)

		// top P
		if topP, ok := promptTemplateFieldValue[float64](b.Config, "TopP"); ok {
			topPValue := topP
			paramsMessagesAPI.TopP = &topPValue
		}

		// For Claude 4.5 models (Sonnet and Haiku), only temperature OR top_p can be specified, not both
		// We keep temperature and set top_p to nil for these models
		if IsClaude45Model(b.Config.ModelID) {
			paramsMessagesAPI.TopP = nil
			logger.Println("Excluding top_p for Claude 4.5 model (only temperature will be used)")
		}

		// top K
		if topK, ok := promptTemplateFieldValue[int](b.Config, "TopK"); ok {
			paramsMessagesAPI.TopK = topK
		}

		// set thinking config for Claude 3.7 if --think flag is enabled
		if !b.Config.Think { // if not set validate if set in prompt template
			if b.Config.PromptTemplate != "" {
				for _, p := range config.Prompts {
					if p.Name == b.Config.PromptTemplate && p.Thinking {
						b.Config.Think = true
					}
				}
			}
		}
		// set text editor config if --text-editor flag is enabled or in prompt template
		if !b.Config.EnableTextEditor { // if not set via CLI flag, check prompt template
			if b.Config.PromptTemplate != "" {
				for _, p := range config.Prompts {
					if p.Name == b.Config.PromptTemplate && p.TextEditor {
						b.Config.EnableTextEditor = true
					}
				}
			}
		}
		logger.Printf("b.Config.Think=%t b.Config.EnableTextEditor=%t b.Config.ModelID=%s", b.Config.Think, b.Config.EnableTextEditor, b.Config.ModelID)

		normalizedModelID := normalizeToModelID(b.Config.ModelID)
		if b.Config.Think && (normalizedModelID == ClaudeV37Sonnet.String() || normalizedModelID == ClaudeV4Sonnet.String() || normalizedModelID == ClaudeV4Opus.String() || normalizedModelID == ClaudeV45Sonnet.String() || normalizedModelID == ClaudeV45Haiku.String()) {
			paramsMessagesAPI.Thinking = NewThinkingConfig()
			logger.Println("enabled thinking feature for Claude 3.7")
			if budget, ok := promptTemplateFieldValue[int](b.Config, "BudgetTokens"); ok {
				paramsMessagesAPI.Thinking.BudgetTokens = budget
			}
			if b.Config.BudgetTokens != 0 { // override with command line flag value if given

				if b.Config.BudgetTokens < mininumThinkingTokens {
					e := fmt.Errorf("%d is less than the minimum budget tokens size of 1024 tokens. Anthropic suggests trying at least 4000 tokens to achieve more comprehensive and nuanced reasoning", b.Config.BudgetTokens)
					return bodsError{e, "BudgetTokens"}
				}
				paramsMessagesAPI.Thinking.BudgetTokens = b.Config.BudgetTokens
			}

		}

		// max tokens
		if maxTokens, ok := promptTemplateFieldValue[int](b.Config, "MaxTokens"); ok {
			paramsMessagesAPI.MaxTokens = maxTokens
		}
		if b.Config.MaxTokens != 0 { // override with command line flag value if given
			paramsMessagesAPI.MaxTokens = b.Config.MaxTokens
		}
		if paramsMessagesAPI.MaxTokens <= b.Config.BudgetTokens {
			e := fmt.Errorf("%d <= %d: Thinking budget tokens must always be less than the max tokens", paramsMessagesAPI.MaxTokens, b.Config.BudgetTokens)
			return bodsError{e, "Tokens"}
		}

		// Add text editor tool if enabled
		textEditorContext := ""
		if b.Config.EnableTextEditor {
			modelID := normalizeToModelID(b.Config.ModelID)
			// Text editor tool is only supported by Claude 3.5v2 Sonnet, Claude 3.7 Sonnet, Claude 4, and Claude 4.5
			if modelID == ClaudeV35SonnetV2.String() || modelID == ClaudeV37Sonnet.String() || modelID == ClaudeV4Sonnet.String() || modelID == ClaudeV4Opus.String() || modelID == ClaudeV45Sonnet.String() || modelID == ClaudeV45Haiku.String() {

				switch {
				case modelID == ClaudeV35SonnetV2.String():
					paramsMessagesAPI.AnthropicBeta = append(paramsMessagesAPI.AnthropicBeta, "computer-use-2024-10-22")
				case (modelID == ClaudeV4Sonnet.String() || modelID == ClaudeV4Opus.String() || modelID == ClaudeV45Sonnet.String() || modelID == ClaudeV45Haiku.String()) && b.Config.Think:
					paramsMessagesAPI.AnthropicBeta = append(paramsMessagesAPI.AnthropicBeta, "interleaved-thinking-2025-05-14")
				default: // for Claude 3.7
					paramsMessagesAPI.AnthropicBeta = append(paramsMessagesAPI.AnthropicBeta, "token-efficient-tools-2025-02-19")
				}

				toolDef := NewTextEditorToolDefinition(modelID)
				paramsMessagesAPI.Tools = append(paramsMessagesAPI.Tools, toolDef)
				logger.Printf("Enabled text editor tool for model %s with tool type %s\n", modelID, toolDef.Type)

				environmentInfo := func() string {
					wd, err := os.Getwd()
					if err != nil {
						return "Error getting working directory: " + err.Error()
					}

					isGitRepo := "No"
					_, err = os.Stat(filepath.Join(wd, ".git"))
					if err == nil {
						isGitRepo = "Yes"
					}

					var sb strings.Builder
					sb.WriteString("\nHere is useful information about the environment you are running in:\n\n<env>\n")
					sb.WriteString(fmt.Sprintf("Working directory: %s\n", wd))
					sb.WriteString(fmt.Sprintf("Is directory a git repo: %s\n", isGitRepo))
					sb.WriteString(fmt.Sprintf("Platform: %s\n", runtime.GOOS))
					sb.WriteString(fmt.Sprintf("Today's date: %s\n", time.Now().Format("1/2/2006")))
					sb.WriteString("</env>\n\n")

					directoryContext := ToolWorkingDirectoryContext()
					logger.Println(directoryContext)
					sb.WriteString(directoryContext)

					return sb.String()
				}
				textEditorContext = environmentInfo()

			} else {
				logger.Printf("Text editor tool is not supported for model %s, ignoring\n", modelID)
			}
		}

		// currently only available for Haiku 3.5 in us-east-2
		// see https://docs.aws.amazon.com/bedrock/latest/userguide/latency-optimized-inference.html
		performanceConfiguration := types.PerformanceConfiguration{Latency: types.PerformanceConfigLatencyStandard}
		if b.Config.ModelID == "us.anthropic.claude-3-5-haiku-20241022-v1:0" && awsConfig.Region == "us-east-2" {
			performanceConfiguration.Latency = types.PerformanceConfigLatencyOptimized
			logger.Println("set performance configuration latency to '", performanceConfiguration.Latency, "'")
		}

		// e.g. echo 'Summarize following text'(=prefix) | bods < file(=content)
		// if a prompt template was given (--prompt) and the template has a 'user'
		// prompt, pre-pend the prefix with the user prompt from the template
		user := ""
		if b.Config.PromptTemplate != "" {
			for _, p := range config.Prompts {
				if p.Name == config.PromptTemplate {
					user = p.User // could be empty TODO
				}
			}
		}

		// replace user prompt input variables e.g. {{.TASK}} with collected input values
		if config.UserPromptInputs != nil {
			var replacedPrompt bytes.Buffer
			tmpl, err := template.New("userPromptTemplate").Parse(user)
			if err != nil {
				panic(err)
			}
			err = tmpl.Execute(&replacedPrompt, b.Config.UserPromptInputs)
			if err != nil {
				panic(err)
			}
			user = replacedPrompt.String()
		}

		// prefix = combined user prompt + Config.Prefix
		// TODO delete prefix := fmt.Sprintf("%s %s", user, b.Config.Prefix)

		// set assistant role from prompt template
		assistant := ""
		if b.Config.PromptTemplate != "" {
			for _, p := range config.Prompts {
				if p.Name == b.Config.PromptTemplate && p.Assistant != "" {
					assistant = p.Assistant
				}
			}
		}
		if b.Config.Assistant != "" { // override if explicitey provided with '--assistant'
			assistant = b.Config.Assistant
		}

		// set system prompt from prompt template, unless system prompt
		// explicitly provided wth '--system'
		logger.Printf("config.PromptTemplate=%s  config.SystemPrompt=%s\n", config.PromptTemplate, config.SystemPrompt)
		if b.Config.PromptTemplate != "" && b.Config.SystemPrompt == "" {
			for _, p := range config.Prompts {
				if p.Name == b.Config.PromptTemplate && p.System != "" {
					b.Config.SystemPrompt = p.System
				}
			}
		}

		// system prompts are currently available for use with Claude 3 models and Claude 2.1
		// for Claude2, system prompt is included in the user prompt
		var system string
		if IsClaude3OrHigherModelID(b.Config.ModelID) {
			paramsMessagesAPI.System = b.Config.SystemPrompt
		} else {
			system = b.Config.SystemPrompt
		}

		// use CRIS is available
		if b.Config.CrossRegionInference {
			inferenceProfilID, err := crossRegionInferenceProfileID(*bedrockClient, b.Config.ModelID, awsConfig.Region)
			if err == nil {
				logger.Println("replacing config.ModelID=", b.Config.ModelID, " with inference profile id ", inferenceProfilID)
				b.Config.ModelID = inferenceProfilID
			}
		}

		format := ""
		if b.Config.Format {
			format = defaultMarkdownFormatText
		} else {
			format = " \n . \n "
		}

		// ORIG LOCATION messages := []Message{{Role: MessageRoleUser}}

		if b.Config.Pasteboard {
			// NEW START
			processor := NewPasteboardProcessor(&config)

			pasteboardContents, err := processor.ProcessPasteboard()
			if err != nil {
				logger.Printf("Error processing pasteboard: %v", err)
				return bodsError{err, "Pasteboard"}
			}

			if len(pasteboardContents) > 0 {
				messages[0].Content = append(messages[0].Content, pasteboardContents...)
				logger.Printf("Added %d content items from pasteboard", len(pasteboardContents))
			}
			// NEW END

			// OLD START
			// t := pasteboard.GetContentType()
			// logger.Printf("pasteboard type=%s", t)
			//
			// if t == MessageContentTypeMediaTypePNG || t == MessageContentTypeMediaTypeGIF || t == MessageContentTypeMediaTypeWEBP {
			// 	if !IsVisionCapable(b.Config.ModelID) {
			// 		e := fmt.Errorf("%s: model does not have vision capability that allows Claude to understand and analyze images", b.Config.ModelID)
			// 		return bodsError{e, "Pasteboard"}
			// 	}
			//
			// 	imgBytes := pasteboard.Read()
			// 	if imgBytes == nil {
			// 		logger.Println("could not read image from pasteboard")
			// 		return bodsError{errors.New("there was a problem reading the image from the clipboard. Did you copy an image to the clipboard?"), "Pasteboard"}
			// 	}
			//
			// 	imgType := http.DetectContentType(imgBytes)
			// 	if !slices.Contains(MessageContentTypes, imgType) {
			// 		panic("unsupported image type " + imgType)
			// 	}
			//
			// 	img, format, err := image.Decode(bytes.NewReader(imgBytes))
			// 	if err != nil {
			// 		panic("could not decode image " + imgType)
			// 	}
			//
			// 	isValidSize, msg := validateImageDimension(img, format)
			// 	if !isValidSize {
			// 		e := fmt.Errorf("%s", msg)
			// 		return bodsError{e, "ImageSize"}
			// 	}
			//
			// 	messages[0].Content = append(messages[0].Content, imgToMessageContent(imgBytes, imgType))
			// }
			// OLD END
		}

		if config.ImageContent != nil {
			logger.Println("adding images from config.ImageContent")
			messages[0].Content = append(messages[0].Content, config.ImageContent...)
		}

		// used replaced content in metaprompt mode
		promptContent := content
		if b.Config.Metamode {
			promptContent = b.Config.Content
		}

		// Build structured content array instead of concatenated string
		// This enables prompt caching by separating logical components into individual content blocks
		// Each component can be cached independently when reused across requests
		var contentBlocks []Content

		// helper function to create a standard user prompt text content
		createUserTextContentWithCaching := func(text string) Content {
			trimmedText := strings.TrimSpace(text)
			c := Content{
				Type: MessageContentTypeText,
				Text: trimmedText,
			}
			// Ensure text field is never empty for text content type
			if c.Text == "" {
				c.Text = " " // Use space to avoid validation errors
			}

			// Only add cache control when text is large enough (at least ~1024 tokens)
			// Claude 3.7 requires at least 1,024 tokens per cache checkpoint
			// Assuming 1 token ≈ 4-5 characters
			// See: https://docs.aws.amazon.com/bedrock/latest/userguide/prompt-caching.html
			minCacheableLength := 5 * 1024 // ~1024 tokens
			if IsPromptCachingSupported(b.Config.ModelID) && len(c.Text) > minCacheableLength {
				c.CacheControl = &CacheControl{
					Type: "ephemeral",
				}
			}
			return c
		}

		// helper function to create a standard user prompt text content
		// createUserPDFContentWithCaching := func(pdf []byte) Content {
		// 	b64Pdf := base64.StdEncoding.EncodeToString(pdf)
		// 	s := Source{
		// 		Type:      "base64",
		// 		MediaType: MessageContentTypeMediaTypePDF, // "application/pdf"
		// 		Data:      b64Pdf,
		// 	}
		// 	pdfContent := Content{
		// 		Type:   MessageContentTypeDocument,
		// 		Source: &s,
		// 		Citations: &Citations{
		// 			Enabled: false,
		// 		},
		// 	}
		//
		// 	if IsPromptCachingSupported(b.Config.ModelID) {
		// 		pdfContent.CacheControl = &CacheControl{
		// 			Type: "ephemeral",
		// 		}
		// 	}
		// 	return pdfContent
		// }

		// 1. System prompt (if present and not using Claude 3+ system field)
		// Only add to content if not handled by paramsMessagesAPI.System
		if system != "" && !IsClaude3OrHigherModelID(b.Config.ModelID) {
			c := Content{
				Type: MessageContentTypeText,
				Text: system,
			}
			contentBlocks = append(contentBlocks, c)
		}

		// 2. Main content (piped input or metamode content)
		if promptContent != "" {

			contentType, _ := getContentTypeFromString(promptContent)

			logger.Printf("getContentTypeFromString(promptContent) = %s\n", contentType)

			promptContentIsImage := false
			if slices.Contains(MessageContentTypes, contentType) && contentType != MessageContentTypeMediaTypePDF {
				imgBytes := []byte(promptContent)
				img, imgType, _ := validateAndDecodeImage(imgBytes)
				if img != nil {
					promptContentIsImage = true
					c := imgToMessageContent(imgBytes, imgType)
					contentBlocks = append(contentBlocks, c)
				}
			}

			// try and extract PDF content in whole sting e.g. from   bods "summarize" < file.pdf
			// See also: https://aws.amazon.com/about-aws/whats-new/2025/06/citations-api-pdf-claude-models-amazon-bedrock/
			// TODO: check Citations API and PDF support for Claude are available for Claude Opus 4, Claude Sonnet 4, Claude Sonnet 3.7, Claude Sonnet 3.5v2.
			var pdfBytes [][]byte
			var promptTextContent string
			if !promptContentIsImage {
				pdfBytes, promptTextContent = ExtractMultiplePDFsFromString(promptContent)
			}

			logger.Printf("extracted %d pdf documents\n", len(pdfBytes))

			if len(pdfBytes) > 0 { // one or more pdfs detected

				for _, pdfData := range pdfBytes {
					if pdfData != nil && validatePDF(pdfData) == nil {

						b64Pdf := base64.StdEncoding.EncodeToString(pdfData)
						s := Source{
							Type:      "base64",
							MediaType: MessageContentTypeMediaTypePDF, // "application/pdf"
							Data:      b64Pdf,
						}
						pdfContent := Content{
							Type:   MessageContentTypeDocument,
							Source: &s,
							Citations: &Citations{
								Enabled: false,
							},
						}

						if IsPromptCachingSupported(b.Config.ModelID) {
							pdfContent.CacheControl = &CacheControl{
								Type: "ephemeral",
							}
						}
						contentBlocks = append(contentBlocks, pdfContent)

					} else { // not a valid pdf document, just append this invalid pdf data as text content
						logger.Printf("no pdf or no valid pdf content, appending invalid pdfData as user prompt text input\n")
						c := createUserTextContentWithCaching(string(pdfData))
						contentBlocks = append(contentBlocks, c)
					}
				}

				// check if 'leftover' is potentially a piped in / redirected image; if not just append text
				if promptTextContent != "" {
					promptTextContent = strings.TrimLeftFunc(promptTextContent, unicode.IsSpace)
					maxChars := min(len(promptTextContent), 100)

					contentType, _ := getContentTypeFromString(promptTextContent)

					logger.Printf("remaining content from pdf extract contentType=%s promptTextContent=%s\n", contentType, promptTextContent[:maxChars])

					promptTextContentIsImage := false
					if slices.Contains(MessageContentTypes, contentType) && contentType != MessageContentTypeMediaTypePDF {
						imgBytes := []byte(promptTextContent)
						img, imgType, _ := validateAndDecodeImage(imgBytes)
						if img != nil {
							promptTextContentIsImage = true
							c := imgToMessageContent(imgBytes, imgType)
							contentBlocks = append(contentBlocks, c)
						}
					}

					if !promptTextContentIsImage && strings.TrimSpace(promptTextContent) != "" {
						logger.Println("appending remaining promptTextConent as user text content")
						c := createUserTextContentWithCaching(promptTextContent)
						contentBlocks = append(contentBlocks, c)
					}
				}
			}

			// else: no PDFs treat as regular text content
			if len(pdfBytes) == 0 && !promptContentIsImage && strings.TrimSpace(promptContent) != "" {
				logger.Println("no PDFs treat as regular text content")
				c := createUserTextContentWithCaching(promptContent)
				contentBlocks = append(contentBlocks, c)
			}
		}

		// 3. User/template prefix (combined user prompt + Config.Prefix)
		// previously above:    prefix := fmt.Sprintf("%s %s", user, b.Config.Prefix)
		if strings.TrimSpace(user) != "" {
			contentBlocks = append(contentBlocks, Content{
				Type: MessageContentTypeText,
				Text: strings.TrimSpace(user),
			})
		}
		if strings.TrimSpace(b.Config.Prefix) != "" {
			contentBlocks = append(contentBlocks, Content{
				Type: MessageContentTypeText,
				Text: strings.TrimSpace(b.Config.Prefix),
			})
		}

		// 4. Text editor context (environment info)
		if strings.TrimSpace(textEditorContext) != "" {
			trimmedText := strings.TrimSpace(textEditorContext)
			c := Content{
				Type: MessageContentTypeText,
				Text: trimmedText,
			}
			// Ensure text field is never empty
			if c.Text == "" {
				c.Text = " " // Use space to avoid validation errors
			}

			// Only add cache control for substantial text (at least ~1024 tokens)
			minCacheableLength := 5 * 1024 // ~1024 tokens
			if IsPromptCachingSupported(b.Config.ModelID) && len(trimmedText) > minCacheableLength {
				c.CacheControl = &CacheControl{
					Type: "ephemeral",
				}
			}
			contentBlocks = append(contentBlocks, c)
		}

		// 5. Format instructions
		if strings.TrimSpace(format) != "" {
			contentBlocks = append(contentBlocks, Content{
				Type: MessageContentTypeText,
				Text: strings.TrimSpace(format),
			})
		}

		// Add all content blocks to the message
		messages[0].Content = append(messages[0].Content, contentBlocks...)

		if strings.TrimSpace(assistant) != "" {
			messages = append(messages,
				Message{
					Role: MessageRoleAssistant,
					Content: []Content{
						{
							Type: MessageContentTypeText,
							Text: strings.TrimRight(assistant, "\n"),
						},
					},
				})
		}

		paramsMessagesAPI.Messages = messages

		body, err := json.Marshal(paramsMessagesAPI)
		if err != nil {
			panic(err)
		}
		// logger.Printf("body=%v\n", spew.Sdump(paramsMessagesAPI))
		logger.Printf("string body=%v\n", string(body))
		if len(os.Getenv("DUMP_PROMPT")) > 0 {
			data, _ := json.MarshalIndent(paramsMessagesAPI, "", "  ")
			fmt.Println(string(data))
			os.Exit(0)
		}

		modelInput := bedrockruntime.InvokeModelWithResponseStreamInput{
			Body:                     body,
			ModelId:                  &b.Config.ModelID,
			ContentType:              aws.String("application/json"),
			Accept:                   aws.String("application/json"),
			PerformanceConfigLatency: performanceConfiguration.Latency,
		}

		modelOutput, err := bedrockRuntimeClient.InvokeModelWithResponseStream(*b.context, &modelInput)
		if err != nil {
			logger.Println(err)
			return bodsError{err, "There was a problem invoking the model. Have you enabled the model and set the correct region?"}
		}

		eventStream := modelOutput.GetStream()

		// Return the completionOutput to be processed by Update
		// before was: 	return b.receiveStreamingMessagesCmd(completionOutput{stream: eventStream})()
		return completionOutput{stream: eventStream}
	}
}

// HandleTextEditorToolResult processes the result from a text editor tool call
func HandleTextEditorToolResult(toolUseID string, result string, isError bool) json.RawMessage {
	response := map[string]any{
		"type":        "tool_result",
		"tool_use_id": toolUseID,
		"content":     result,
	}
	if isError {
		response["is_error"] = true
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		logger.Printf("Error marshaling tool result: %v\n", err)
		return nil
	}

	return responseJSON
}

// invokeModelForToolResponse handles invoking the model after a tool response and returns a tea.Msg
// see also https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking#example-passing-thinking-blocks-with-tool-results
func (b *Bods) invokeModelForToolResponse() tea.Msg {
	paramsMessagesAPI.Messages = messages

	// not working on Bedrock (yet): https://docs.anthropic.com/en/docs/build-with-claude/tool-use/token-efficient-tool-use

	// prepare the model input
	body, err := json.Marshal(paramsMessagesAPI)
	if err != nil {
		panic(err)
	}

	if len(os.Getenv("DEBUG")) > 0 { // don't marshal if debug not set
		data, _ := json.MarshalIndent(paramsMessagesAPI, "", "  ")
		logger.Printf("InvokeModelWithResponseStreamInput:\n%s\n", string(data))
	}

	// Implement exponential backoff with jitter for ThrottlingException handling
	const maxRetries = 6
	const baseDelay = 2 * time.Second
	var modelOutput *bedrockruntime.InvokeModelWithResponseStreamOutput
	var lastError error

	for attempt := range maxRetries {

		modelInput := bedrockruntime.InvokeModelWithResponseStreamInput{
			Body:        body,
			ModelId:     &b.Config.ModelID,
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
		}

		if attempt > 0 {
			// Calculate backoff with jitter for retries; exponential backoff: baseDelay * 2^attempt
			// Add jitter: random value between 0 and 1 second
			jitter := time.Duration(rand.Int63n(1000)) * time.Millisecond // #nosec G404 - Weak random is acceptable for jitter
			delay := baseDelay*(1<<attempt) + jitter
			logger.Printf("Retrying API call (attempt %d/%d) after %v delay due to throttling", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		modelOutput, err = bedrockRuntimeClient.InvokeModelWithResponseStream(*b.context, &modelInput)

		if err == nil {
			break // success, break out of retry loop
		}

		lastError = err
		logger.Printf("API call attempt %d failed: %v", attempt+1, err)

		// Check if error is a ThrottlingException
		if strings.Contains(err.Error(), "ThrottlingException") {
			// Continue to next retry
			logger.Println("Detected ThrottlingException, will retry with backoff")
			continue
		} else {
			// For non-throttling errors, don't retry
			break
		}
	}

	// If we exhausted all retries or had a different error
	if err != nil {
		logger.Println(lastError)
		return bodsError{lastError, "There was a problem invoking the model. Have you enabled the model and set the correct region?"}
	}

	eventStream := modelOutput.GetStream()

	// return the new stream as output to be processed by Update
	return completionOutput{stream: eventStream}
}

func (b *Bods) receiveStreamingMessagesCmd(msg completionOutput) tea.Cmd {
	// logger.Printf("receiveStreamingMessagesCmd msg.stream=%v\n", msg.stream)
	return func() tea.Msg {
		var stopReason string
		const timeSleep = 30 * time.Millisecond
		for {
			select {
			case responseStream := <-msg.stream.Reader.Events():
				logger.Printf("responseStream=%s\n", responseStream)

				if responseStream == nil {
					return bodsError{errEmptyResponseStream, "The response stream was empty (nil)."}
				}

				switch v := responseStream.(type) {
				case *types.ResponseStreamMemberChunk: // logger.Printf("ResponseStreamMemberChunk [v.Value.Bytes]: %s\n", v.Value.Bytes)
					var msgResponse AnthropicClaudeMessagesResponse
					err := json.Unmarshal(v.Value.Bytes, &msgResponse)
					if err != nil {
						panic(err)
					}

					logger.Printf("msgResponse.Type: %s\n", msgResponse.Type)

					if msgResponse.Type == EventMessageStart.String() {
						// {"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": "message", "role": "assistant", "content": [], "model": "claude-3-7-sonnet-20250219", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 25, "output_tokens": 1}}}

						logger.Printf("event: message_start role=%s id=%s\n", msgResponse.Message.Role, msgResponse.Message.ID)
						if msgResponse.Message.Role == MessageRoleAssistant {
							messages = append(messages,
								Message{
									Role:    MessageRoleAssistant,
									Content: []Content{},
								})
						}

					}

					// event: message_stop
					if msgResponse.Type == EventMessageStop.String() {

						logger.Println("event: message_stop")

						if stopReason == MessageContentTypeToolUse {
							toolCallInputByte := []byte(b.Config.ToolCallJSONString)
							editorResult := HandleTextEditorToolCall(toolCallInputByte)

							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1

							// create tool response message
							messages = append(messages,
								Message{
									Role: MessageRoleUser,
									Content: []Content{{
										Type:      "tool_result",
										ToolUseID: messages[lastMsgIdx].Content[lastContentIdx].ID,
										Content:   editorResult.Content,
									}},
								})

							// return a special message that will trigger a new model invocation
							_ = msg.stream.Close()
							return b.invokeModelForToolResponse()
						}

						logger.Printf("type:message_stop v.Value.Bytes=%s\n", v.Value.Bytes)
						logger.Printf("type:message_stop outputTokenCount=%d\n", msgResponse.AmazonBedrockInvocationMetrics.OutputTokenCount)
						_ = msg.stream.Close()
						msg.stream = nil
						msg.content = ""
						return msg
					}

					//
					// content_block_start
					//
					if msgResponse.Type == EventContentBlockStart.String() {

						// currentRole := messages[len(messages)-1].Role

						msg.content = ""
						if msgResponse.ContentBlock.Type == "thinking" && b.Config.Format {
							msg.content = "`<thinking>` \n\n"
						}

						if msgResponse.ContentBlock.Type == "text" { // && currentRole == MessageRoleAssistant {
							logger.Println("content_block_start type='text'")
							messages[len(messages)-1].Content = append(messages[len(messages)-1].Content,
								Content{
									Type: MessageContentTypeText,
									Text: " ", // Use space to avoid validation errors for empty text
								})
						}

						if msgResponse.ContentBlock.Type == "tool_use" {
							logger.Println("content_block_start type='tool_use'")
							// {{{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_bdrk_01NHfgPyKd23Dy57k97Rn2ou","name":"str_replace_editor","input":{}}} {}} {}}
							// DEBUG msg.content = fmt.Sprintf("\n\nSTART tool_use id=%s name=%s\n", msgResponse.ContentBlock.ID, msgResponse.ContentBlock.Name)
							messages[len(messages)-1].Content = append(messages[len(messages)-1].Content,
								Content{
									Type: MessageContentTypeToolUse,
									ID:   msgResponse.ContentBlock.ID,
									Name: msgResponse.ContentBlock.Name,
								})
							b.Config.ToolCallJSONString = "" // reset to empty string
						}

						if msgResponse.ContentBlock.Type == "thinking" {
							logger.Println("content_block_start type='thinking'")
							messages[len(messages)-1].Content = append(messages[len(messages)-1].Content,
								Content{
									Type:      MessageContentTypeThinking, // "type": "thinking",
									Thinking:  "",                         // "thinking": "Let me analyze this step by step...",
									Signature: "",                         // "signature": "WaU..."
								})
						}

						return msg

					} // content_block_start END

					//
					// content_block_delta START
					//
					if msgResponse.Type == EventContentBlockDelta.String() {
						msg.isThinkingOutput = false // default to no thinking output

						// type can be thinking | thinking_delta | text | text_delta | signature_delta
						if msgResponse.Delta.Type == "thinking_delta" {
							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1
							messages[lastMsgIdx].Content[lastContentIdx].Thinking += msgResponse.Delta.Thinking

							msg.content = msgResponse.Delta.Thinking
							msg.isThinkingOutput = true
							return msg
						}

						if msgResponse.Delta.Type == "signature_delta" {
							logger.Printf("signature_delta=%s", msgResponse.Delta.Signature)
							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1
							messages[lastMsgIdx].Content[lastContentIdx].Signature += msgResponse.Delta.Signature

							// if msgResponse.ContentBlock.Type == "text" && b.Config.Think && b.Config.Format {
							if b.Config.Think && b.Config.Format {
								msg.content = "\n\n`</thinking>`\n\n"
							}
							return msg
						}

						// debug [59429] responseStream=&{{{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":""}} {}} {}}
						if msgResponse.Delta.Type == "input_json_delta" {
							// you can accumulate the string deltas and parse the JSON once you receive a content_block_stop
							b.Config.ToolCallJSONString = fmt.Sprintf("%s%s", b.Config.ToolCallJSONString, msgResponse.Delta.PartialJSON)
							msg.content = ""
							return msg
						}

						if msgResponse.Delta.Type == "text_delta" {
							role := messages[len(messages)-1].Role
							if role != MessageRoleAssistant {
								logger.Printf("ERROR should be assistant role %v\n", messages)
							}

							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1
							messages[lastMsgIdx].Content[lastContentIdx].Text += msgResponse.Delta.Text

							// DEL t := messages[len(messages)-1].Content[0].Text
							// DEL messages[len(messages)-1].Content[0].Text = t + msgResponse.Delta.Text
						}

						msg.content = msgResponse.Delta.Text
						return msg
					} // content_block_delta END

					if msgResponse.Type == EventContentBlockStop.String() { //nolint:staticcheck // SA9003: empty branch
						// debug [62732] responseStream=&{{{"type":"content_block_stop","index":1} {}} {}}
					}

					// debug [55908] responseStream=&{{{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":116}} {}} {}}
					if msgResponse.Type == EventMessageDelta.String() {
						stopReason = msgResponse.Delta.StopReason
						if stopReason == "tool_use" {
							logger.Println("b.Config.ToolCallJSONString=" + b.Config.ToolCallJSONString)

							toolCallInputByte := []byte(b.Config.ToolCallJSONString)

							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1
							messages[lastMsgIdx].Content[lastContentIdx].Input = toolCallInputByte
						}
					}

					logger.Printf("WARN ignoring response type '%s'", msgResponse.Type)

				default:
					logger.Printf("receiveStreamingMessagesCmd - sleeping (switch) for %dms; v = %v", timeSleep, v) // XX
					time.Sleep(timeSleep)
				}
			case <-(*b.context).Done():
				_ = msg.stream.Close()
				return bodsError{errContextCanceled, "The context was cancelled."}
			default:
				time.Sleep(timeSleep)
			}
		}
	}
}

// readStdinCmd reads from stdin and returns a tea.Msg wrapping the content read.
func readStdinCmd() tea.Msg {
	logger.Printf("readStdInCmd: isInputTerminal=%v\n", isInputTerminal())
	if !isInputTerminal() {
		reader := bufio.NewReader(os.Stdin)
		stdinBytes, err := io.ReadAll(reader)
		if err != nil {
			return bodsError{err, "Unable to read from stdin."}
		}

		// debug output
		maxLength := min(len(stdinBytes), 100)
		logger.Printf("DEBUG readStdinCmd len=%d \n%s\n", len(stdinBytes), string(stdinBytes[:maxLength]))

		return promptInput{string(stdinBytes) + " "}
	}
	return promptInput{""} // used to be " " as hack so string is not empty
}

// readStdin reads from stdin and returns the content read as string.
func readStdin() (string, error) {
	logger.Printf("readStdIn: isInputTerminal=%v\n", isInputTerminal())
	if !isInputTerminal() {
		reader := bufio.NewReader(os.Stdin)
		stdinBytes, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("unable to read from stdin")
		}
		// debug output
		maxLength := min(len(string(stdinBytes)), 50)
		logger.Printf("DEBUG readStdin len=%d: %s\n", len(stdinBytes), string(stdinBytes)[:maxLength])
		return string(stdinBytes), nil
	}
	return "", nil
}

func initialBodsModel(r *lipgloss.Renderer, cfg *Config) *Bods {
	ctx, cancel := context.WithCancel(context.Background())
	glamRenderer, _ := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(100), // wrap output at specific width (default is 80)
		glamour.WithAutoStyle(),   // detect bg color and pick either the default dark or light theme
	)

	return &Bods{
		Styles:        makeStyles(r),
		Config:        cfg,
		cancelRequest: cancel,
		state:         startState,
		context:       &ctx,
		glam:          glamRenderer,
	}
}

// promptInput a tea.Msg wrapping the content read from stdin.
type promptInput struct {
	content string
	// inputType string // 'text' or 'image'
}

// completionOutput a tea.Msg that wraps the content returned.
type completionOutput struct {
	content          string
	isThinkingOutput bool
	stream           *bedrockruntime.InvokeModelWithResponseStreamEventStream
}

// -------------------
// styles.go
// -------------------
type styles struct {
	AppName,
	CliArgs,
	Comment,
	CyclingChars,
	ErrorHeader,
	ErrorDetails,
	ErrPadding,
	Flag,
	FlagComma,
	FlagDesc,
	InlineCode,
	Link,
	Pipe,
	Quote,
	ConversationList,
	SHA1,
	Bullet,
	Timeago lipgloss.Style
}

func makeStyles(r *lipgloss.Renderer) (s styles) {
	const horizontalEdgePadding = 2
	s.AppName = r.NewStyle().Bold(true)
	s.CliArgs = r.NewStyle().Foreground(lipgloss.Color("#585858"))
	s.Comment = r.NewStyle().Foreground(lipgloss.Color("#757575"))
	s.CyclingChars = r.NewStyle().Foreground(lipgloss.Color("#FF87D7"))
	s.ErrorHeader = r.NewStyle().Foreground(lipgloss.Color("#F1F1F1")).Background(lipgloss.Color("#A33D56")).Bold(true).Padding(0, 1).SetString("ERROR")
	s.ErrorDetails = s.Comment
	s.ErrPadding = r.NewStyle().Padding(0, horizontalEdgePadding)
	s.Flag = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#00B594", Dark: "#3EEFCF"}).Bold(true)
	s.FlagComma = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#5DD6C0", Dark: "#427C72"}).SetString(",")
	s.FlagDesc = s.Comment
	s.InlineCode = r.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Background(lipgloss.Color("#3A3A3A")).Padding(0, 1)
	s.Link = r.NewStyle().Foreground(lipgloss.Color("#00AF87")).Underline(true)
	s.Quote = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#FF71D0", Dark: "#FF78D2"})
	s.Pipe = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8470FF", Dark: "#745CFF"})
	s.ConversationList = r.NewStyle().Padding(0, 1)
	s.SHA1 = s.Flag
	s.Bullet = r.NewStyle().SetString("• ").Foreground(lipgloss.AdaptiveColor{Light: "#757575", Dark: "#777"})
	s.Timeago = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#555"})
	return s
}

func extractXMLTagContent(xmlContent string, tag string) string {
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"
	logger.Printf("extracting content of xml tag %s%s\n", startTag, endTag)
	start := strings.Index(xmlContent, startTag)
	if start == -1 {
		return ""
	}
	start += len(startTag)
	end := strings.Index(xmlContent[start:], endTag)
	if end == -1 {
		return ""
	}
	end += start
	return xmlContent[start:end]
}

func parseVarMap(input string) (map[string]string, error) {
	vars := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, "file://") {
			filename := value[7:]
			filename = strings.Trim(filename, `"`)
			filename = strings.Trim(filename, `'`)
			data, err := os.ReadFile(filename)
			if err == nil {
				value = string(data)
			} else {
				return nil, err
			}
		} else {
			value = strings.Trim(value, `"`)
			value = strings.Trim(value, `'`)
		}
		name = strings.ToUpper(name)
		if value != "" {
			vars[name] = value
		}
	}
	return vars, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
