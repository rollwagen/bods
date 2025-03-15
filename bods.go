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
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"text/template"
	"time"

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
	"github.com/rollwagen/bods/pasteboard"
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
	logger.Printf("Update() msg.content=%s\n", msg)
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
	logger.Printf("startMessagesCmd: len(content)=%d\n", len(content)) // content is piped input e.g. echo "content" | bods

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
			b.Config.ModelID = ClaudeV37Sonnet.String()
		}
		logger.Println("config.ModelID set to: ", b.Config.ModelID)

		// top P
		if topP, ok := promptTemplateFieldValue[float64](b.Config, "TopP"); ok {
			paramsMessagesAPI.TopP = topP
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
		logger.Printf("b.Config.Think=%t  b.Config.ModelID=%s", b.Config.Think, b.Config.ModelID)

		normalizedModelID := normalizeToModelID(b.Config.ModelID)
		if b.Config.Think && (normalizedModelID == ClaudeV37Sonnet.String() || normalizedModelID == ClaudeV4Sonnet.String()) {
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
			// Text editor tool is only supported by Claude 3.5 Sonnet and Claude 3.7 Sonnet
			if modelID == ClaudeV35Sonnet.String() || modelID == ClaudeV35SonnetV2.String() || modelID == ClaudeV37Sonnet.String() {
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
		prefix := fmt.Sprintf("%s %s", user, b.Config.Prefix)

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

		// get image from pasteboard
		if b.Config.Pasteboard {

			if !IsVisionCapable(b.Config.ModelID) {
				e := fmt.Errorf("%s: model does not have vision capability that allows Claude to understand and analyze images", b.Config.ModelID)
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

		// used replaced content in metaprompt mode
		promptContent := content
		if b.Config.Metamode {
			promptContent = b.Config.Content
		}
		textContent := Content{
			Type: MessageContentTypeText,
			Text: fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n", system, prefix, promptContent, textEditorContext, format),
		}
		messages[0].Content = append(messages[0].Content, textContent)

		if assistant != "" {
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

		return b.receiveStreamingMessagesCmd(completionOutput{stream: eventStream})()
	}
}

// HandleTextEditorToolResult processes the result from a text editor tool call
func HandleTextEditorToolResult(ctx context.Context, toolUseID string, result string, isError bool) json.RawMessage {
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

func (b *Bods) receiveStreamingMessagesCmd(msg completionOutput) tea.Cmd {
	// logger.Printf("receiveStreamingMessagesCmd msg.stream=%v\n", msg.stream)
	return func() tea.Msg {
		const timeSleep = 20 * time.Millisecond
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
						// {"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": "message", "role":
						//   "assistant", "content": [], "model": "claude-3-7-sonnet-20250219", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 25, "output_tokens": 1}}}

						logger.Println("\n------ MESSAGE_START: " + msgResponse.Message.Role)
						if msgResponse.Message.Role == MessageRoleAssistant {
							messages = append(messages,
								Message{
									Role:    MessageRoleAssistant,
									Content: []Content{},
								})
						}

					}

					if msgResponse.Type == EventMessageStop.String() {
						logger.Printf("------ MESSAGE_STOP ------ %v", messages[len(messages)-1])

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

						currentRole := messages[len(messages)-1].Role

						msg.content = ""
						if msgResponse.ContentBlock.Type == "thinking" && b.Config.Format {
							msg.content = "`<thinking>` \n\n"
						}

						if msgResponse.ContentBlock.Type == "text" && b.Config.Think && b.Config.Format {
							msg.content = "\n\n`</thinking>`\n\n"
						}

						if msgResponse.ContentBlock.Type == "text" && currentRole == MessageRoleAssistant {
							messages[len(messages)-1].Content = append(messages[len(messages)-1].Content,
								Content{
									Type: MessageContentTypeText,
									Text: "",
								})
						}

						if msgResponse.ContentBlock.Type == "tool_use" {
							// {{{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_bdrk_01NHfgPyKd23Dy57k97Rn2ou","name":"str_replace_editor","input":{}}} {}} {}}
							// logger.Printf("tool_use: %s", msgResponse.Message)
							msg.content = fmt.Sprintf("\n\nSTART tool_use id=%s name=%s\n", msgResponse.ContentBlock.ID, msgResponse.ContentBlock.Name) // DEBUG
							messages[len(messages)-1].Content = append(messages[len(messages)-1].Content,
								Content{
									Type: MessageContentTypeToolUse,
									ID:   msgResponse.ContentBlock.ID,
									Name: msgResponse.ContentBlock.Name,
								})
							b.Config.ToolCallJSONString = "" // reset to empty string
						}

						return msg

					} // content_block_start

					if msgResponse.Type == EventContentBlockDelta.String() {
						msg.isThinkingOutput = false // default to no thinking output
						// type can be thinking | thinking_delta | text | text_delta
						if msgResponse.Delta.Type == "thinking_delta" {
							msg.content = msgResponse.Delta.Thinking
							msg.isThinkingOutput = true
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
							t := messages[len(messages)-1].Content[0].Text
							messages[len(messages)-1].Content[0].Text = t + msgResponse.Delta.Text
						}

						msg.content = msgResponse.Delta.Text
						return msg
					}

					if msgResponse.Type == EventContentBlockStop.String() {
						// debug [62732] responseStream=&{{{"type":"content_block_stop","index":1} {}} {}}
					}

					// debug [55908] responseStream=&{{{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":116}} {}} {}}
					if msgResponse.Type == EventMessageDelta.String() {
						if msgResponse.Delta.StopReason == "tool_use" {
							logger.Println("---- message_delta -- delta -- stop_reason:tool_use ---- \n" + b.Config.ToolCallJSONString)

							toolCallInputByte := []byte(b.Config.ToolCallJSONString)
							// toolCallInputString := b.Config.ToolCallJSONString

							lastMsgIdx := len(messages) - 1
							lastContentIdx := len(messages[lastMsgIdx].Content) - 1
							logger.Println(toolCallInputByte)
							messages[lastMsgIdx].Content[lastContentIdx].Input = toolCallInputByte
							logger.Println(b.Config.ToolCallJSONString)
							logger.Println(messages[lastMsgIdx])

							editorResult, editorErr := HandleTextEditorToolCall(*b.context, toolCallInputByte)
							if editorErr != nil {
								logger.Printf("ERROR %v\n", editorErr)
							}

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

							// { "role": "user", "content": [
							//          {
							//              "type": "tool_result",
							//              "tool_use_id": "toolu_01A09q90qw90lq917835lq9", # from the API response
							//              "content": "65 degrees" # from running your tool
							//          } ] }

							paramsMessagesAPI.Messages = messages

							body, err := json.Marshal(paramsMessagesAPI)
							if err != nil {
								panic(err)
							}

							data, _ := json.MarshalIndent(paramsMessagesAPI, "", "  ")
							logger.Println(string(data))

							modelInput := bedrockruntime.InvokeModelWithResponseStreamInput{
								Body:        body,
								ModelId:     &b.Config.ModelID,
								ContentType: aws.String("application/json"),
								Accept:      aws.String("application/json"),
								// PerformanceConfigLatency: performanceConfiguration.Latency,
							}

							modelOutput, err := bedrockRuntimeClient.InvokeModelWithResponseStream(*b.context, &modelInput)
							if err != nil {
								logger.Println(err)
								return bodsError{err, "There was a problem invoking the model. Have you enabled the model and set the correct region?"}
							}

							eventStream := modelOutput.GetStream()

							return b.receiveStreamingMessagesCmd(completionOutput{stream: eventStream})()

							//
							//
							//

							if err != nil {
								_ = msg.stream.Close()
								return bodsError{errContextCanceled, "The context was cancelled."}
							}
							msg.content = "\n\nRESULT:" + editorResult.Content
							return msg
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
		logger.Printf("DEBUG readStdinCmd len=%d: %s\n", len(stdinBytes), string(stdinBytes))
		return promptInput{string(stdinBytes) + " "}
	}
	return promptInput{" "} // hack so string is not empty
}

// readStdin reads from stdin and returns the content read as string.
func readStdin() (string, error) {
	logger.Printf("readStdInCmd: isInputTerminal=%v\n", isInputTerminal())
	if !isInputTerminal() {
		reader := bufio.NewReader(os.Stdin)
		stdinBytes, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("unable to read from stdin")
		}
		logger.Printf("DEBUG readStdin len=%d: %s\n", len(stdinBytes), string(stdinBytes))
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
