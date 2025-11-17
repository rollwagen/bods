# CLAUDE.md - AI Assistant Guide for `bods`

This document provides comprehensive information about the `bods` codebase for AI assistants to understand and work effectively with this project.

## Project Overview

**Name:** bods
**Description:** A CLI tool for interacting with Anthropic's Claude models via AWS Bedrock
**Language:** Go 1.23+
**Primary Framework:** Bubble Tea (TUI), Cobra (CLI)
**License:** MIT

`bods` is inspired by [mods](https://github.com/charmbracelet/mods) but specifically designed for Claude models on Amazon Bedrock. It supports Unix-style piping, file redirection, clipboard integration (macOS), and pre-configured prompt templates.

### Key Features

- AWS Bedrock integration for Claude models (v2, v3, v3.5, v3.7, v4)
- Streaming responses with real-time rendering
- Clipboard image support (macOS only, via Objective-C bridge)
- Pre-configured prompt templates with autocomplete
- Metaprompt mode for generating optimized prompts
- Cross-region inference profile (CRIS) support
- Extended thinking support for Claude 3.7+ models
- XML tag content extraction
- Template variable system for dynamic prompts

## Repository Structure

```
bods/
├── main.go                      # CLI entry point, flag handling, Cobra setup
├── bods.go                      # Core Bubble Tea model, AWS Bedrock logic
├── config.go                    # Configuration management, YAML parsing
├── types.go                     # Anthropic API type definitions
├── cross_region.go              # Cross-region inference profile handling
├── terminal.go                  # Terminal/TTY detection utilities
├── bods.yaml                    # Embedded prompt templates (578 lines)
├── config_test.go               # Configuration tests
├── pasteboard/                  # Platform-specific clipboard support
│   ├── pasteboard.go            # Interface definition
│   ├── pasteboard_darwin.go     # macOS implementation (Objective-C)
│   ├── pasteboard_darwin.m      # Objective-C bridge
│   ├── pasteboard_linux.go      # Linux stub
│   └── pasteboard_windows.go    # Windows stub
├── assets/                      # Build assets
│   ├── cosign/                  # Code signing keys
│   └── scripts/                 # Build scripts (completions.sh)
├── .github/workflows/           # CI/CD workflows
│   ├── release.yml              # Release automation
│   └── codeql.yml               # Security analysis
├── .goreleaser.yaml             # GoReleaser configuration
├── .golangci.yml                # Linter configuration
├── .pre-commit-config.yaml      # Pre-commit hooks
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── release.sh                   # Release script
└── README.md                    # User documentation
```

### Architecture Pattern

**Single-package design:** The project uses a flat structure (not cmd/pkg/internal) as it's a straightforward CLI tool. This keeps the codebase simple and accessible.

## Key Components and Architecture

### 1. Entry Point (`main.go`)

**Responsibilities:**
- Initialize Cobra CLI framework
- Register flags with autocomplete
- Load configuration (embedded or user override)
- Handle errors with styled output
- Set version from build info

**Key Functions:**
- `init()` - Sets up debug logging, version info
- `main()` - Entry point, loads config, executes command
- `ensureConfig()` - Loads embedded `bods.yaml` or user override from `$XDG_CONFIG_HOME/bods/bods.yaml`
- `initFlags()` - Registers all CLI flags with Cobra
- `handleError()` - Custom error formatting with lipgloss styling

### 2. Core Logic (`bods.go`)

**Architecture:** Elm Architecture via Bubble Tea

**State Machine:**
```go
const (
    startState      // Initial state - setup
    doneState       // Terminal state - completion
    requestState    // Making AWS Bedrock API call
    responseState   // Processing streaming response
    errorState      // Error handling state
)
```

**Bubble Tea Methods:**
- `Init()` - Read stdin, parse template variables
- `Update(tea.Msg)` - State transitions, message handling
- `View() string` - Render current state (with Glamour markdown)

**AWS Bedrock Integration:**
- `startMessagesCmd()` - Invoke Bedrock with streaming
- `receiveStreamingMessagesCmd()` - Process streaming chunks
- Uses `bedrockruntime.Client.InvokeModelWithResponseStream()`
- Supports all Claude model types (Sonnet, Opus, Haiku, Claude 4)

**Special Features:**
- **Thinking Mode:** Claude 3.7+ extended thinking with token budgets
- **Image Support:** Multimodal with clipboard images (macOS)
- **Streaming:** Real-time response rendering
- **XML Extraction:** Write specific XML tag content to files

### 3. Configuration System (`config.go`)

**Configuration Hierarchy:**
1. Embedded `bods.yaml` (compiled into binary)
2. User override at `$XDG_CONFIG_HOME/bods/bods.yaml`
3. CLI flags (highest priority)
4. Environment variables (DEBUG, DUMP_PROMPT)

**Config Structure:**
```go
type Config struct {
    Prefix               string              // User query
    ModelID              string              // Claude model ID
    SystemPrompt         string              // System prompt
    Assistant            string              // Assistant prefill
    Prompts              []Prompt            // Loaded templates
    PromptTemplate       string              // Selected template
    UserPromptInputs     map[string]string   // Template variables
    MaxTokens            int                 // Token limit
    Format               bool                // Markdown formatting
    Metamode             bool                // Metaprompt mode
    Pasteboard           bool                // Clipboard integration
    CrossRegionInference bool                // Use CRIS
    Think                bool                // Extended thinking
    BudgetTokens         int                 // Thinking budget
}
```

**Prompt Template Format:**
```yaml
prompts:
  template-name:
    description: "Human-readable description for autocomplete"
    model_id: anthropic.claude-3-sonnet-20240229-v1:0
    max_tokens: 1000
    temperature: 1.0
    top_p: 0.999
    top_k: 250
    thinking: true              # Enable for Claude 3.7+
    budget_tokens: 4096         # Thinking token budget
    system: |
      System prompt text
    user: |
      User prompt with {{.VARIABLES}}
    assistant: |
      Assistant prefill (optional)
```

**Template Variables:**
- Go template syntax: `{{.VARIABLE_NAME}}`
- Metaprompt mode rewrites: `{$VAR}` → `{{.VAR}}`
- Interactive prompting or `--variable-input` flag
- File input support: `VARIABLE=file://path.txt`

### 4. Type Definitions (`types.go`)

Defines complete Anthropic Messages API types for AWS Bedrock:

**Key Types:**
- `MessageRequest` - Request payload
- `MessageResponse` - Response payload
- `Content` - Content blocks (text/image)
- `ImageContent` - Image data with base64 encoding
- `Message` - User/assistant message
- `ThinkingBlock` - Claude 3.7+ thinking output
- `TextBlock` - Standard text response
- `StreamEvent` - Streaming response events

**Supported Model IDs:**
```go
const (
    ClaudeV2         = "anthropic.claude-v2"
    ClaudeV2_1       = "anthropic.claude-v2:1"
    ClaudeV3Sonnet   = "anthropic.claude-3-sonnet-20240229-v1:0"
    ClaudeV3Opus     = "anthropic.claude-3-opus-20240229-v1:0"
    ClaudeV3Haiku    = "anthropic.claude-3-haiku-20240307-v1:0"
    ClaudeV35Sonnet  = "anthropic.claude-3-5-sonnet-20240620-v1:0"
    ClaudeV37Sonnet  = "anthropic.claude-3-7-sonnet-20250219"
    ClaudeV4Sonnet   = "anthropic.claude-4-sonnet-20250514"
)
```

### 5. Cross-Region Inference (`cross_region.go`)

**Purpose:** Automatically discover and use AWS Bedrock CRIS profiles for better availability and performance.

**Key Features:**
- Caches discovered profiles in BuntDB (embedded key-value store)
- Cache location: `$XDG_DATA_HOME/bods/cris.db`
- 24-hour TTL on cached profiles
- Falls back to direct model invocation if CRIS unavailable

**Functions:**
- `getCachedCRISProfileArn()` - Retrieve cached profile
- `discoverCRISProfile()` - Call Bedrock API to find profile
- `cacheProfileArn()` - Store profile in BuntDB

### 6. Platform-Specific Code (`pasteboard/`)

**Clipboard Integration:**
- **macOS (darwin):** Full support via Objective-C bridge
  - `pasteboard_darwin.go` - Go interface
  - `pasteboard_darwin.m` - Objective-C implementation
  - Uses CGO to access NSPasteboard APIs
  - Supports PNG, JPEG, TIFF formats
- **Linux:** Stub implementation (not supported)
- **Windows:** Stub implementation (not supported)

**Build Tags:**
```go
//go:build darwin
//go:build linux
//go:build windows
```

## Development Workflow

### Local Setup

```bash
# Clone repository
git clone https://github.com/rollwagen/bods.git
cd bods

# Download dependencies
go mod download

# Build
go build -o bods

# Install locally (optional)
go install

# Run with debug logging
DEBUG=1 ./bods "your prompt"

# Dump request without API call
DUMP_PROMPT=1 ./bods "your prompt"
```

### Development Branch Strategy

- **Main branch:** `main` (default)
- **Feature branches:** Use descriptive names (e.g., `feature/add-claude-4-support`)
- **Release tags:** Semantic versioning (e.g., `v1.0.0`)

### Making Changes

#### Adding a New Prompt Template

1. Edit `bods.yaml` in the repository root
2. Add new entry under `prompts:` section
3. Rebuild to embed the new configuration
4. Test with `./bods --prompt your-new-prompt`

**Example:**
```yaml
prompts:
  code-review:
    description: "Review code for bugs and improvements"
    model_id: anthropic.claude-3-5-sonnet-20240620-v1:0
    max_tokens: 2000
    system: |
      You are an expert code reviewer. Focus on:
      - Bugs and edge cases
      - Performance issues
      - Security vulnerabilities
      - Code style and best practices
    user: |
      Review the following code:
      {{.CODE}}
```

#### Adding a New CLI Flag

1. Add field to `Config` struct in `config.go`
2. Register flag in `initFlags()` in `main.go`
3. Handle flag logic in `bods.go` or relevant file
4. Update autocomplete if needed
5. Add tests

**Example:**
```go
// In config.go
type Config struct {
    // ... existing fields
    NewFeature bool `koanf:"new_feature"`
}

// In main.go initFlags()
rootCmd.Flags().BoolVar(&globalConfig.NewFeature,
    "new-feature", false, "Enable new feature")
```

#### Adding Support for a New Model

1. Add constant to `types.go`:
```go
const ClaudeNewModel = "anthropic.claude-new-model-id"
```

2. Update model selection logic in `bods.go` if needed
3. Test with `--model` flag
4. Update documentation in README.md

### Testing

**Current State:** Minimal test coverage (room for improvement)

**Running Tests:**
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestEnsureConfig
```

**Writing Tests:**
- Use `github.com/stretchr/testify/assert` for assertions
- Place test files next to source files (`*_test.go`)
- Mock AWS SDK calls for unit tests (currently lacking)

**Test Coverage Goals:**
- Config loading and parsing
- Template variable substitution
- AWS Bedrock request construction
- Streaming response parsing
- Error handling

### Code Quality Tools

#### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linters
golangci-lint run

# Run with fixes
golangci-lint run --fix
```

**Enabled Linters:** (see `.golangci.yml`)
- `bodyclose` - HTTP response body closing
- `errcheck` - Unchecked errors
- `goconst` - Repeated strings
- `gocritic` - Comprehensive checks
- `gosec` - Security issues
- `govet` - Go vet checks
- `ineffassign` - Ineffectual assignments
- `revive` - Fast, extensible linter
- `staticcheck` - Advanced static analysis
- `unconvert` - Unnecessary conversions
- `unparam` - Unused parameters
- `unused` - Unused code

#### Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

**Hooks Enabled:** (see `.pre-commit-config.yaml`)
- `go-imports` - Import formatting
- `go-mod-tidy` - Dependency cleanup
- `gofumpt` - Stricter formatting than gofmt
- `golangci-lint` - Linting
- `commitlint` - Conventional commits
- YAML validation

#### Formatting

```bash
# Format with gofumpt (stricter than gofmt)
gofumpt -l -w .

# Standard go fmt
go fmt ./...

# Organize imports
goimports -w .
```

### Debugging

#### Debug Logging

Enable comprehensive debug logging:

```bash
DEBUG=1 ./bods "your prompt"
# Output: DEBUG logging to file /tmp/bods.log
```

Debug log includes:
- Configuration loading
- Template parsing
- AWS SDK calls
- Streaming events
- State transitions

**Warning:** Debug logs contain full prompts and responses!

#### Dump Prompt

See the exact JSON sent to AWS Bedrock without making the API call:

```bash
DUMP_PROMPT=1 ./bods "hello"
```

Output:
```json
{
  "anthropic_version": "bedrock-2023-05-31",
  "messages": [...],
  "system": "...",
  "temperature": 1,
  "max_tokens": 200,
  "top_p": 0.999
}
```

#### Show Configuration

```bash
./bods --show-config
```

Shows loaded configuration including all prompt templates.

## Build and Release Process

### Manual Build

```bash
# Simple build
go build -o bods

# Build with version info
go build -ldflags "-s -w -X main.build=v1.0.0" -o bods

# Build for specific platform
GOOS=darwin GOARCH=arm64 go build -o bods-darwin-arm64
```

### Release Process

**Automated via GitHub Actions:**

1. **Create Release:**
```bash
# Use release script
./release.sh v1.2.3

# Or manually
git tag v1.2.3
git push origin v1.2.3
```

2. **GitHub Actions Workflow:** (`.github/workflows/release.yml`)
   - Triggers on tag push
   - Runs on `macos-latest` (required for CGO)
   - Sets up Go 1.21+
   - Installs cosign for binary signing
   - Runs GoReleaser

3. **GoReleaser Process:** (`.goreleaser.yaml`)
   - Pre-build: Generate shell completions
   - Build: macOS binaries (amd64, arm64) with CGO
   - Sign: Cosign signatures for checksums
   - Archive: Include README, LICENSE, completions
   - Publish: GitHub Releases
   - Homebrew: Update `rollwagen/homebrew-tap`

**Build Requirements:**
- **CGO_ENABLED=1** - Required for pasteboard Objective-C bridge
- **macOS runner** - Required for CGO cross-compilation
- **Cosign key** - For code signing (secret: `COSIGN_PWD`)
- **GitHub token** - For releases (secret: `GORELEASER_TOKEN`)

**Release Artifacts:**
- Binary: `bods` (macOS amd64, arm64)
- Completions: bash, zsh, fish
- Checksums: `checksums.txt`
- Signatures: Cosign signatures
- Archive: `.tar.gz` with docs and completions

### Version Management

**Version Setting:**
- Set via `ldflags` during build: `-X main.build={{.Version}}`
- Displayed in `--version` flag
- Semver format required (e.g., `v1.2.3`)

**Changelog:**
- Auto-generated from commit messages
- Excludes `docs:` and `test:` commits
- Sorted ascending by commit

## Configuration System

### Configuration Files

**Embedded Default:**
- File: `bods.yaml` (compiled into binary)
- Uses `//go:embed` directive
- Always available, no setup required

**User Override:**
- Location: `$XDG_CONFIG_HOME/bods/bods.yaml`
  - Typically: `~/.config/bods/bods.yaml` (Linux/macOS)
  - Windows: `%APPDATA%\bods\bods.yaml`
- Replaces embedded config entirely (not merged)
- Created manually by user

**Checking Active Config:**
```bash
./bods --show-config
```

### Prompt Template System

**Template Features:**
- Pre-configured prompts with descriptions
- Per-prompt model selection
- Parameter overrides (tokens, temperature, etc.)
- System and user prompt sections
- Assistant prefill for response control
- Template variables with Go template syntax

**Using Templates:**
```bash
# List available prompts (via autocomplete)
./bods --prompt <TAB><TAB>

# Use specific template
./bods --prompt summarize < document.txt

# Override template settings
./bods --prompt summarize --model anthropic.claude-3-opus-20240229-v1:0 -t 4000
```

**Template Variables:**

Example template with variables:
```yaml
code-explain:
  user: |
    Explain this {{.LANGUAGE}} code:
    {{.CODE}}
```

Usage:
```bash
# Interactive mode (will prompt for LANGUAGE and CODE)
./bods --prompt code-explain

# Non-interactive with variable input
./bods --prompt code-explain --variable-input LANGUAGE=Python,CODE=file://script.py

# Or with metaprompt mode
cat generated_prompt.txt | ./bods --metaprompt-mode --variable-input VAR1=value1
```

### Metaprompt Mode

**Purpose:** Use Anthropic's metaprompt to generate optimized prompts

**Workflow:**
```bash
# 1. Generate prompt with metaprompt
bods --prompt metaprompt --format=false --tag-content Instructions -t 4096
# Task: Review AWS CloudFormation templates for security

# 2. Use generated prompt (saved in Instructions.txt)
cat Instructions.txt | bods --metaprompt-mode --variable-input TEMPLATE=file://stack.yaml
```

**Metaprompt Mode Features:**
- Rewrites `{$VARIABLE}` → `{{.VARIABLE}}` automatically
- Interactive variable prompting
- File input support: `VAR=file://path.txt`
- Handles Anthropic metaprompt format

### Model Selection

**Default:** Claude 3 Sonnet (`anthropic.claude-3-sonnet-20240229-v1:0`)

**Override Methods:**
1. CLI flag: `--model anthropic.claude-3-opus-20240229-v1:0`
2. Prompt template: `model_id` field in YAML
3. Config file: Default model setting (if added)

**Available Models:**
- `anthropic.claude-v2` - Claude 2
- `anthropic.claude-v2:1` - Claude 2.1
- `anthropic.claude-3-haiku-20240307-v1:0` - Claude 3 Haiku
- `anthropic.claude-3-sonnet-20240229-v1:0` - Claude 3 Sonnet
- `anthropic.claude-3-opus-20240229-v1:0` - Claude 3 Opus
- `anthropic.claude-3-5-sonnet-20240620-v1:0` - Claude 3.5 Sonnet
- `anthropic.claude-3-7-sonnet-20250219` - Claude 3.7 Sonnet (thinking support)
- `anthropic.claude-4-sonnet-20250514` - Claude 4 Sonnet (thinking support)

## Code Style and Conventions

### Go Style Guidelines

**Standards:**
- Go 1.23+ features and idioms
- `gofumpt` formatting (stricter than `gofmt`)
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) where applicable

**Naming Conventions:**
- Exported types/functions: `PascalCase`
- Unexported types/functions: `camelCase`
- Constants: `PascalCase` for exported, `camelCase` for internal
- Acronyms: All caps (e.g., `HTTPServer`, `URLPath`)

**Error Handling:**
```go
// Custom error type with context
type bodsError struct {
    err    error
    reason string
}

// Always wrap errors with context
if err != nil {
    return bodsError{err: err, reason: "failed to load config"}
}
```

**Concurrency Patterns:**
```go
// Use sync.OnceValue for lazy initialization
var isInputTerminal = sync.OnceValue(func() bool {
    return isatty.IsTerminal(os.Stdin.Fd())
})

// Bubble Tea commands for async operations
func (b *Bods) fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        // Long-running operation
        return dataFetchedMsg{data: result}
    }
}
```

### Architecture Patterns

#### Elm Architecture (Bubble Tea)

```go
// Model holds state
type Bods struct {
    state      string
    response   string
    err        error
    // ... other fields
}

// Init returns initial command
func (b Bods) Init() tea.Cmd {
    return readStdinCmd
}

// Update handles messages and state transitions
func (b Bods) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle input
    case completionOutput:
        // Handle async result
    }
    return b, nil
}

// View renders current state
func (b Bods) View() string {
    if b.state == doneState {
        return b.response
    }
    return "Processing..."
}
```

#### Command Pattern

```go
// Commands are functions that return messages
type tea.Cmd func() tea.Msg

// Execute async operation
func fetchDataCmd() tea.Cmd {
    return func() tea.Msg {
        data, err := fetchData()
        return dataMsg{data: data, err: err}
    }
}

// Chain commands
func (b Bods) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case startMsg:
        return b, tea.Batch(
            firstCmd(),
            secondCmd(),
        )
    }
}
```

#### Embedded Resources

```go
//go:embed bods.yaml
var bodsConfig []byte

// Use in code
config, err := yaml.Parse(bodsConfig)
```

#### Build Tags

```go
//go:build darwin

package pasteboard

// macOS-specific implementation
```

### File Organization

**Single Package Structure:**
- All main code in root package
- Subpackages for platform-specific code only
- Keep related functions in same file
- Separate concerns by file (config.go, types.go, etc.)

**File Naming:**
- `*_test.go` - Test files
- `*_darwin.go` - macOS-specific
- `*_linux.go` - Linux-specific
- `*_windows.go` - Windows-specific

## Common Tasks and Patterns

### Reading User Input

**From stdin (piped or redirected):**
```go
func readStdinCmd() tea.Msg {
    if !isInputTerminal() {
        data, err := io.ReadAll(os.Stdin)
        if err != nil {
            return stdinMsg{err: err}
        }
        return stdinMsg{content: string(data)}
    }
    return stdinMsg{}
}
```

**Interactive prompts (using huh):**
```go
import "github.com/charmbracelet/huh"

form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("Enter value").
            Value(&userInput),
    ),
)
err := form.Run()
```

### AWS Bedrock Invocation

**Streaming Request:**
```go
bedrockClient := bedrockruntime.NewFromConfig(cfg)

request := &MessageRequest{
    AnthropicVersion: "bedrock-2023-05-31",
    Messages: []Message{
        {Role: "user", Content: []Content{{Type: "text", Text: prompt}}},
    },
    MaxTokens:   1000,
    Temperature: 1.0,
}

requestJSON, _ := json.Marshal(request)

output, err := bedrockClient.InvokeModelWithResponseStream(ctx,
    &bedrockruntime.InvokeModelWithResponseStreamInput{
        ModelId:     aws.String(modelID),
        Body:        requestJSON,
        ContentType: aws.String("application/json"),
    },
)

// Process streaming events
stream := output.GetStream()
events := stream.Events()
for event := range events {
    switch e := event.(type) {
    case *types.ResponseStreamMemberChunk:
        // Process chunk
    }
}
```

**With CRIS Profile:**
```go
profileArn, _ := getCachedCRISProfileArn(ctx, modelID, bedrockClient)
if profileArn != "" {
    modelID = profileArn
}
```

### Template Variable Processing

**Parse template variables:**
```go
import "text/template"

tmpl, err := template.New("prompt").Parse(promptText)
if err != nil {
    return err
}

// Execute with data
var buf bytes.Buffer
err = tmpl.Execute(&buf, data)
```

**Metaprompt mode rewriting:**
```go
// Convert {$VARIABLE} to {{.VARIABLE}}
re := regexp.MustCompile(`\{\$([A-Z_]+)\}`)
converted := re.ReplaceAllString(input, "{{.$1}}")
```

### Markdown Rendering

```go
import "github.com/charmbracelet/glamour"

renderer, err := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(80),
)

output, err := renderer.Render(markdownText)
```

### XML Tag Extraction

```go
import "regexp"

tagName := "Instructions"
pattern := fmt.Sprintf("<%s>(.*?)</%s>", tagName, tagName)
re := regexp.MustCompile(pattern)

matches := re.FindStringSubmatch(response)
if len(matches) > 1 {
    content := matches[1]
    os.WriteFile(tagName+".txt", []byte(content), 0644)
}
```

### Clipboard Image (macOS)

```go
import "github.com/rollwagen/bods/pasteboard"

imageData, err := pasteboard.GetImageBytes()
if err != nil {
    return err
}

base64Image := base64.StdEncoding.EncodeToString(imageData)

content := Content{
    Type: "image",
    Source: &ImageContent{
        Type:      "base64",
        MediaType: "image/png",
        Data:      base64Image,
    },
}
```

## Platform-Specific Considerations

### macOS (darwin)

**Full Support:**
- Clipboard image integration via NSPasteboard
- CGO required for Objective-C bridge
- Supported architectures: amd64, arm64

**Build Requirements:**
```bash
CGO_ENABLED=1 go build
```

**Clipboard Implementation:**
- Uses Objective-C runtime via CGO
- Supports PNG, JPEG, TIFF formats
- Requires AppKit framework

### Linux

**Support Status:**
- Core functionality: Full support
- Clipboard images: Not supported (stub implementation)

**No CGO Required:**
```bash
CGO_ENABLED=0 go build
```

**Potential Enhancement:**
- Could add X11/Wayland clipboard support in future

### Windows

**Support Status:**
- Core functionality: Full support
- Clipboard images: Not supported (stub implementation)

**No CGO Required:**
```bash
CGO_ENABLED=0 go build
```

**Potential Enhancement:**
- Could add Win32 clipboard API support in future

### Cross-Platform Code

**Platform-Specific Files:**
```
pasteboard_darwin.go   // +build darwin
pasteboard_linux.go    // +build linux
pasteboard_windows.go  // +build windows
```

**Stub Implementations:**
```go
//go:build linux

package pasteboard

func GetImageBytes() ([]byte, error) {
    return nil, errors.New("clipboard not supported on Linux")
}
```

## Dependencies

### Core Dependencies

**AWS SDK:**
- `github.com/aws/aws-sdk-go-v2` - AWS SDK base
- `github.com/aws/aws-sdk-go-v2/config` - Configuration
- `github.com/aws/aws-sdk-go-v2/service/bedrock` - Bedrock management
- `github.com/aws/aws-sdk-go-v2/service/bedrockruntime` - Bedrock inference

**UI Framework:**
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/glamour` - Markdown rendering
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/huh` - Interactive forms

**CLI:**
- `github.com/spf13/cobra` - CLI framework

**Configuration:**
- `github.com/knadh/koanf/v2` - Config management
- `github.com/goccy/go-yaml` - YAML parsing
- `github.com/adrg/xdg` - XDG directories

**Storage:**
- `github.com/tidwall/buntdb` - Embedded key-value DB

**Utilities:**
- `github.com/mattn/go-isatty` - TTY detection
- `github.com/muesli/termenv` - Terminal capabilities
- `github.com/fatih/color` - Colored output

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get -u github.com/charmbracelet/bubbletea@latest

# Verify dependencies
go mod verify
```

## Important Notes for AI Assistants

### AWS Bedrock vs Direct API

**Critical:** This project uses **AWS Bedrock**, NOT the direct Anthropic API.

**Differences:**
- Endpoint: AWS Bedrock regional endpoints (e.g., us-east-1)
- Authentication: AWS credentials (IAM, not Anthropic API keys)
- Model IDs: AWS Bedrock format (e.g., `anthropic.claude-3-sonnet-20240229-v1:0`)
- API: AWS SDK with `InvokeModelWithResponseStream`
- Permissions: Requires Bedrock model access granted in AWS account

**Never suggest:**
- Using Anthropic API keys
- Direct `api.anthropic.com` endpoints
- Anthropic SDK (use AWS SDK instead)

### CGO Dependency

**Important:** macOS builds require CGO for clipboard support.

**Implications:**
- Cannot use `CGO_ENABLED=0` for macOS builds
- Cross-compilation is complex (use macOS runners)
- Linux/Windows builds don't need CGO (clipboard not supported)

**When working on pasteboard code:**
- Test on actual macOS machine
- Objective-C knowledge helpful
- Consider impact on build process

### Streaming is Required

**All responses use streaming API:**
- Not optional, core to architecture
- Bubble Tea handles async streaming naturally
- Don't suggest non-streaming alternatives

**Why streaming:**
- Better UX (real-time feedback)
- Handles long responses well
- Natural fit for Bubble Tea

### Configuration System

**Embedded config is replaced, not merged:**
- User `bods.yaml` completely replaces embedded version
- Not merged or overlaid
- Document this clearly to users

**Template variables:**
- Use Go template syntax: `{{.VAR}}`
- Metaprompt mode uses: `{$VAR}` (auto-converted)
- Don't mix syntaxes

### Testing Gaps

**Current state:** Minimal test coverage

**When adding features:**
- Add tests for new functionality
- Consider integration tests for AWS SDK calls
- Mock AWS SDK responses for unit tests
- Test error handling paths

**Priority areas for testing:**
- Config parsing
- Template variable substitution
- AWS request construction
- Streaming response parsing
- Error handling

### Version Compatibility

**Go version:** 1.23+ required

**Features used:**
- `sync.OnceValue` (Go 1.21+)
- Modern generics (Go 1.18+)
- Clear built-in (Go 1.21+)

**When updating Go version:**
- Update `go.mod`
- Update CI workflows
- Update GoReleaser config
- Test on all platforms

### Error Handling Best Practices

**Custom errors:**
```go
type bodsError struct {
    err    error
    reason string
}
```

**Always provide context:**
- User-friendly reason
- Underlying technical error
- Actionable guidance

**Example:**
```go
if err != nil {
    return bodsError{
        err:    err,
        reason: "Failed to load configuration. Check ~/.config/bods/bods.yaml",
    }
}
```

### Security Considerations

**Debug logging:**
- Contains full prompts and responses
- Sensitive data may be logged
- Warn users about debug logs

**AWS credentials:**
- Never log credentials
- Use AWS SDK credential chain
- Document credential requirements

**User input:**
- Sanitize for XML injection (tag extraction)
- Validate template variable names
- Check file paths for file:// input

### Performance Considerations

**CRIS caching:**
- Reduces API calls
- 24-hour TTL
- BuntDB is lightweight
- Cache location: `$XDG_DATA_HOME/bods/cris.db`

**Streaming:**
- Process chunks incrementally
- Don't buffer entire response
- Memory efficient for large responses

**Embedded config:**
- No disk I/O for default config
- Fast startup
- Single binary distribution

### User Experience

**Markdown rendering:**
- Use Glamour for consistent styling
- Auto-detect terminal capabilities
- Word wrapping for readability

**Autocomplete:**
- Essential for discoverability
- Generate completions in build process
- Support bash, zsh, fish

**Error messages:**
- Clear, actionable guidance
- Styled with lipgloss
- Separate technical details from user message

## Common Development Scenarios

### Adding a New Feature

1. **Plan the feature:**
   - Identify affected files
   - Consider configuration needs
   - Plan testing approach

2. **Implement:**
   - Add code in appropriate files
   - Follow existing patterns
   - Handle errors gracefully

3. **Configure:**
   - Add config fields if needed
   - Update `bods.yaml` if applicable
   - Update flag registration

4. **Test:**
   - Write unit tests
   - Test manually with `DEBUG=1`
   - Test with `DUMP_PROMPT=1`
   - Test all platforms if applicable

5. **Document:**
   - Update README.md
   - Update this CLAUDE.md
   - Add code comments

6. **Release:**
   - Commit with conventional commit message
   - Pre-commit hooks will validate
   - Create release when ready

### Debugging Issues

1. **Enable debug logging:**
   ```bash
   DEBUG=1 ./bods "test prompt"
   tail -f /tmp/bods.log
   ```

2. **Check request:**
   ```bash
   DUMP_PROMPT=1 ./bods "test prompt"
   ```

3. **Verify config:**
   ```bash
   ./bods --show-config
   ```

4. **Test AWS credentials:**
   ```bash
   aws bedrock list-foundation-models --region us-east-1
   ```

5. **Check model access:**
   ```bash
   aws bedrock get-foundation-model \
     --model-identifier anthropic.claude-3-sonnet-20240229-v1:0 \
     --region us-east-1
   ```

### Working with Prompt Templates

1. **Create new template:**
   - Edit `bods.yaml`
   - Add under `prompts:` section
   - Include description for autocomplete

2. **Test template:**
   ```bash
   ./bods --prompt your-template "test input"
   ```

3. **Test with variables:**
   ```bash
   ./bods --prompt your-template --variable-input VAR1=value1,VAR2=value2
   ```

4. **Rebuild to embed:**
   ```bash
   go build -o bods
   ```

### Updating Dependencies

1. **Check for updates:**
   ```bash
   go list -u -m all
   ```

2. **Update:**
   ```bash
   go get -u ./...
   go mod tidy
   ```

3. **Test:**
   ```bash
   go test ./...
   ./bods --version
   ```

4. **Commit:**
   ```bash
   git add go.mod go.sum
   git commit -m "chore: update dependencies"
   ```

## Quick Reference

### Useful Commands

```bash
# Build
go build -o bods

# Run tests
go test ./...

# Lint
golangci-lint run

# Format
gofumpt -l -w .

# Debug
DEBUG=1 ./bods "prompt"

# Dump request
DUMP_PROMPT=1 ./bods "prompt"

# Show config
./bods --show-config

# Generate completions
./bods completion zsh > _bods
./bods completion bash > bods.bash
./bods completion fish > bods.fish

# Release
./release.sh v1.2.3
```

### File Locations

```
~/.config/bods/bods.yaml          # User config
~/.local/share/bods/cris.db       # CRIS cache
/tmp/bods.log                     # Debug log
```

### Key Files to Understand

1. `main.go` - Entry point, CLI setup
2. `bods.go` - Core logic, AWS integration
3. `config.go` - Configuration system
4. `types.go` - API types
5. `bods.yaml` - Prompt templates
6. `.goreleaser.yaml` - Release config

### Environment Variables

```bash
DEBUG=1                # Enable debug logging
DUMP_PROMPT=1          # Dump request JSON and exit
AWS_REGION=us-east-1   # AWS region
AWS_PROFILE=default    # AWS profile
```

## Contributing Guidelines

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for Claude 4
fix: correct streaming response parsing
docs: update README with new examples
chore: update dependencies
test: add config loading tests
```

### Pull Request Process

1. Create feature branch
2. Make changes with tests
3. Run pre-commit hooks
4. Create PR with description
5. Wait for CI to pass
6. Address review feedback

### Code Review Focus

- Correctness
- Error handling
- Test coverage
- Documentation
- Performance
- Security
- User experience

## Resources

### Documentation

- [AWS Bedrock Documentation](https://docs.aws.amazon.com/bedrock/)
- [Anthropic Claude Documentation](https://docs.anthropic.com/)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Cobra Documentation](https://cobra.dev/)

### Similar Projects

- [mods](https://github.com/charmbracelet/mods) - Inspiration for bods
- [aichat](https://github.com/sigoden/aichat) - AI chat CLI
- [fabric](https://github.com/danielmiessler/fabric) - Prompt patterns

### Support

- GitHub Issues: https://github.com/rollwagen/bods/issues
- GitHub Discussions: https://github.com/rollwagen/bods/discussions

---

**Last Updated:** 2025-11-17
**Document Version:** 1.0.0
**Repository:** https://github.com/rollwagen/bods
