package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// _ "x/image/webp"

	"github.com/aws/aws-sdk-go-v2/aws"
	sdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rollwagen/bods/pasteboard"
)

var errContextCanceled = errors.New("context was canceled")

type state int

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
	return nil
}

func (b *Bods) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if msg.content != "" {
			logger.Printf("completionOutput content=%s\n", msg.content)
			b.Output += msg.content
			if isOutputTerminal() {
				b.glamOutput, _ = b.glam.Render(b.Output)
			}
			b.state = responseState
		}
		if msg.stream == nil {
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
		if b.Config.ShowSettings {
			_ = printConfig(isOutputTerminal())
			b.state = doneState
			return b, b.quit
		}
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
	logger.Printf("startMessagesCmd: len(content)=%d\n", len(content))

	return func() tea.Msg {
		paramsMessagesAPI := NewAnthropicClaudeMessagesInferenceParameters()

		const defaultMarkdownFormatText = " Format the response as markdown without enclosing backticks."

		// use model as specified in prompt template, unless overridden with '--model' flag
		promptTemplateModelID, _ := promptTemplateFieldValue[string](b.Config, "ModelID")
		if b.Config.ModelID == "" && promptTemplateModelID != "" {
			b.Config.ModelID = promptTemplateModelID
		}
		if b.Config.ModelID == "" { // initialize to default if no modelID given at all
			b.Config.ModelID = ClaudeV3Haiku.String()
		}
		logger.Println("config.ModelID set to: ", b.Config.ModelID)

		if topP, ok := promptTemplateFieldValue[float64](b.Config, "TopP"); ok {
			paramsMessagesAPI.TopP = topP
		}

		if topK, ok := promptTemplateFieldValue[int](b.Config, "TopK"); ok {
			paramsMessagesAPI.TopK = topK
		}

		if maxTokens, ok := promptTemplateFieldValue[int](b.Config, "MaxTokens"); ok {
			paramsMessagesAPI.MaxTokens = maxTokens
		}
		if b.Config.MaxTokens != 0 { // override with command line flag value if given
			paramsMessagesAPI.MaxTokens = b.Config.MaxTokens
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
		// prefix = combined user prompt + Config.Prefix
		prefix := fmt.Sprintf("%s %s", user, b.Config.Prefix)

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
		if b.Config.ModelID == ClaudeV21.String() || b.Config.ModelID == ClaudeV3Sonnet.String() {
			paramsMessagesAPI.System = b.Config.SystemPrompt
		} else {
			system = b.Config.SystemPrompt
		}

		format := ""
		if b.Config.Format {
			format = defaultMarkdownFormatText
		}

		messages := []Message{{Role: MessageRoleUser}}

		// get image from pasteboard
		if b.Config.Pasteboard {

			if !IsClaude3ModelID(b.Config.ModelID) {
				e := fmt.Errorf("%s: only Claude3 models have vision capabilities that allow Claude to understand and analyze images", b.Config.ModelID)
				return bodsError{e, "Pasteboard"}
			}

			imgBytes := pasteboard.Read()
			if imgBytes == nil {
				logger.Println("could not read image from pasteboard")
				return bodsError{errors.New("there was a problem reading the image from the clipboard. Did you copy an image to the clipboard?"), "Pasteboard"}
			}

			imgType := http.DetectContentType(imgBytes)
			if !slices.Contains(MessageContentTypes, imgType) {
				panic("unsupported image type " + imgType)
			}

			img, format, err := image.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				panic("could not decode image " + imgType)
			}
			// max width of an image is 8000 pixels; see https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages.html
			const maxSizeImage = 8000
			x := img.Bounds().Max.X
			y := img.Bounds().Max.Y
			logger.Printf("%d x %d size of image\n", x, y)
			if img.Bounds().Max.X > maxSizeImage || img.Bounds().Max.Y > maxSizeImage {
				e := fmt.Errorf("the maximum height and width of an image is %d pixels. %s has size %d x %d", maxSizeImage, format, x, y)
				return bodsError{e, "ImageSize"}
			}

			b64Img := base64.StdEncoding.EncodeToString(imgBytes)
			s := Source{
				Type:      "base64",
				MediaType: imgType,
				Data:      b64Img,
			}
			imageContent := Content{
				Type:   MessageContentTypeImage,
				Source: &s,
			}
			messages[0].Content = append(messages[0].Content, imageContent)
		}

		textContent := Content{
			Type: MessageContentTypeText,
			Text: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n", system, prefix, content, format),
		}
		messages[0].Content = append(messages[0].Content, textContent)

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
			Body:        body,
			ModelId:     &b.Config.ModelID,
			ContentType: aws.String("application/json"),
			Accept:      aws.String("application/json"),
		}

		cfg, err := sdkconfig.LoadDefaultConfig(*b.context)
		if err != nil {
			msg := fmt.Sprintf("LoadDefaultConfig(): failed to load SDK configuration, %v", err)
			log.Fatalf(msg)
		}

		br := bedrockruntime.NewFromConfig(cfg)
		modelOutput, err := br.InvokeModelWithResponseStream(*b.context, &modelInput)
		if err != nil {
			logger.Println(err)
			return bodsError{err, "There was a problem invoking the model. Have you enabled the model and set the correct region?"}
		}

		eventStream := modelOutput.GetStream()

		return b.receiveStreamingMessagesCmd(completionOutput{stream: eventStream})()
	}
}

func (b *Bods) receiveStreamingMessagesCmd(msg completionOutput) tea.Cmd {
	logger.Printf("receiveStreamingMessagesCmd msg.stream=%v\n", msg.stream)
	return func() tea.Msg {
		const timeSleep = 20 * time.Millisecond
		for {
			select {
			case responseStream := <-msg.stream.Reader.Events():
				// logger.Printf("responseStream=%v\n", responseStream) // XX
				switch v := responseStream.(type) {
				case *types.ResponseStreamMemberChunk:
					logger.Printf("ResponseStreamMemberChunk [v.Value.Bytes]: %s\n", v.Value.Bytes)
					var msgResponse AnthropicClaudeMessagesResponse //  new
					err := json.Unmarshal(v.Value.Bytes, &msgResponse)
					if err != nil {
						panic(err)
					}

					if msgResponse.Type == EventMessageStop.String() {
						logger.Printf("type:message_stop outputTokenCount=%d\n", msgResponse.AmazonBedrockInvocationMetrics.OutputTokenCount)
						_ = msg.stream.Close()
						msg.stream = nil
						msg.content = ""
						return msg
					}

					if msgResponse.Type == EventContentBlockDelta.String() {
						msg.content = msgResponse.Delta.Text
						return msg
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
		logger.Printf("DEBUG readStdinCmd len=%d: %s\n", len(stdinBytes), string(stdinBytes))
		return promptInput{string(stdinBytes)}
	}
	return promptInput{""}
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
	content string
	stream  *bedrockruntime.InvokeModelWithResponseStreamEventStream
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
	s.ErrorDetails = s.Comment.Copy()
	s.ErrPadding = r.NewStyle().Padding(0, horizontalEdgePadding)
	s.Flag = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#00B594", Dark: "#3EEFCF"}).Bold(true)
	s.FlagComma = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#5DD6C0", Dark: "#427C72"}).SetString(",")
	s.FlagDesc = s.Comment.Copy()
	s.InlineCode = r.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Background(lipgloss.Color("#3A3A3A")).Padding(0, 1)
	s.Link = r.NewStyle().Foreground(lipgloss.Color("#00AF87")).Underline(true)
	s.Quote = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#FF71D0", Dark: "#FF78D2"})
	s.Pipe = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#8470FF", Dark: "#745CFF"})
	s.ConversationList = r.NewStyle().Padding(0, 1)
	s.SHA1 = s.Flag.Copy()
	s.Bullet = r.NewStyle().SetString("• ").Foreground(lipgloss.AdaptiveColor{Light: "#757575", Dark: "#777"})
	s.Timeago = r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#999", Dark: "#555"})
	return s
}
