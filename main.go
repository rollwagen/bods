package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"text/template"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	// "github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// Build vars.
var (
	//nolint: gochecknoglobals
	Version   = ""
	CommitSHA = "111111111r1111111115b9e7c1df111111b326d2" // TODO
)

func buildVersion() {
	if len(CommitSHA) > 0 {
		vt := rootCmd.VersionTemplate()
		rootCmd.SetVersionTemplate(vt[:len(vt)-1] + " (" + CommitSHA[0:7] + ")\n")
	}
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	rootCmd.Version = Version
}

func init() {
	debugEnabled := len(os.Getenv("DEBUG")) > 0
	if debugEnabled {
		path := filepath.Join(os.TempDir(), "bods.log")

		// Clear log file if it exists and is not empty
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			if err := os.Truncate(path, 0); err != nil {
				fmt.Printf("warning: could not clear log file: %v\n", err)
			}
		}

		f, err := tea.LogToFile(path, "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		// defer f.Close()

		_, _ = fmt.Fprintf(os.Stderr, "DEBUG logging to file %s\n", path)

		prefix := fmt.Sprintf("debug [%d] ", os.Getpid())
		logger = log.New(f, prefix, log.Lmsgprefix)

		logger.Println(time.Now().Format("2006-01-02 15:04:05.000000") + " starting mods...")
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	// glamour.DarkStyleConfig.CodeBlock.Chroma.Error.BackgroundColor = new(string)
	// glamour.LightStyleConfig.CodeBlock.Chroma.Error.BackgroundColor = new(string)

	buildVersion()

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

var (
	config  Config
	program *tea.Program

	logger *log.Logger

	rootCmd = &cobra.Command{
		Use:           "bods",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			config.Prefix = strings.Join(args, " ")
			logger.Println("main.go config.Prefix: " + config.Prefix)

			opts := []tea.ProgramOption{
				// tea.WithOutput(stderrRenderer().Output()),
				tea.WithOutput(os.Stderr),
			}

			if !isInputTerminal() {
				opts = append(opts, tea.WithInput(nil)) // To disable input entirely pass nil
			}

			bods := initialBodsModel(stderrRenderer(), &config)

			// get values for Go template vars by asking input from user
			askTemplateVarInputs := func(templateText string) (map[string]string, error) {
				inputs := make(map[string]string)
				re := regexp.MustCompile(`\{\{\.([a-zA-Z_]+)\}\}`) // e.g. {{.TASK}}
				matches := re.FindAllStringSubmatch(templateText, -1)

				for _, templateVariable := range matches {
					_, ok := inputs[templateVariable[1]]
					if ok {
						continue // only need to capture input for each var once, if it used multiple times
					}
					templateInput := ""
					err := huh.NewForm(
						huh.NewGroup(
							huh.NewText().
								Title(fmt.Sprintf("Input for %s", templateVariable[1])).
								Value(&templateInput)),
					).Run()
					if err != nil {
						return nil, fmt.Errorf("text input failed")
					}
					inputs[templateVariable[1]] = templateInput
				}
				return inputs, nil
			}

			user := ""
			if config.PromptTemplate != "" {
				for _, p := range config.Prompts {
					if p.Name == config.PromptTemplate {
						user = p.User // could be empty TODO
					}
				}
				logger.Printf("prompt template '%s' = %s\n", config.PromptTemplate, _max100Chars(user))
			}

			if user != "" {
				userPromptInputs, err := askTemplateVarInputs(user)
				if err != nil {
					return bodsError{err: err, reason: "Text input failed."}
				}
				config.UserPromptInputs = userPromptInputs
			}

			if config.Metamode {
				stdin, _ := readStdin()
				// rewrite variables: {$F_A_Q} and {$VARIABLE} => {{.F_A_Q}}} and {{.VARIABLE}}}
				rewriteVars := func(input string) string {
					re := regexp.MustCompile(`\{(\$[A-Z_]+)\}`)
					return re.ReplaceAllStringFunc(input, func(match string) string {
						removedPrefix := strings.TrimPrefix(match, "{$")
						removedPrefixSuffix := strings.ToUpper(strings.TrimSuffix(removedPrefix, "}"))
						return fmt.Sprintf("{{.%s}}", removedPrefixSuffix)
					})
				}
				rewrittenStdin := rewriteVars(stdin)

				var inputVariablesValues map[string]string
				var err error
				if config.VariableInputRaw == "" { // aks for variable values interactively, ask not provided as param
					inputVariablesValues, err = askTemplateVarInputs(rewrittenStdin)
					if err != nil {
						return bodsError{err: err, reason: "Text input failed."}
					}
				} else {
					inputVariablesValues, err = parseVarMap(config.VariableInputRaw)
					if err != nil {
						return bodsError{err: err, reason: "Input variables parsing failed."}
					}
				}

				var replacedVarsStdin bytes.Buffer
				tmpl, err := template.New("stdinTemplate").Parse(rewrittenStdin)
				if err != nil {
					panic(err)
				}
				err = tmpl.Execute(&replacedVarsStdin, inputVariablesValues)
				if err != nil {
					panic(err)
				}
				config.Content = replacedVarsStdin.String()
			}

			if config.ImagesFlagInput != "" {
				logger.Println("parsing images flag content...")
				imageContents, err := parseImageURLList(config.ImagesFlagInput)
				if err != nil {
					return bodsError{err, "Error processing content of --images flag"}
				}

				logger.Printf("after parsing imageConent=%v\n", imageContents)

				config.ImageContent = imageContents
			}

			if config.ShowSettings {
				_ = printConfig(isOutputTerminal())
				os.Exit(0)
			}

			logger.Println("creating new tea program...")
			program = tea.NewProgram(bods, opts...)
			logger.Println("running new tea program...")
			m, err := program.Run()
			if err != nil {
				return bodsError{err, "Could not start Bubble Tea program."}
			}

			bods = m.(*Bods)
			if bods.Error != nil {
				return *bods.Error
			}

			if isOutputTerminal() {
				logger.Println("rendering output... isOutputTerminal() == true")
				switch {
				case bods.glamOutput != "":
					fmt.Print(bods.glamOutput)
				case bods.Output != "":
					fmt.Print(bods.Output)
				}
			} else {
				logger.Printf("rendering output... isOutputTerminal() == false -- bods.Output=%s\n", bods.Output)
				if bods.Output != "" {
					fmt.Print(bods.Output)
				}
			}

			return nil
		},
	}
)

func initFlags() {
	const (
		flagModel          = "model"     // the specific model to use
		flagSystem         = "system"    // system prompt
		flagAssistant      = "assistant" // assistant role as part of prompt
		flagPrompt         = "prompt"    // prompt name (template) to use
		flagMaxTokens      = "tokens"    // max nr of tokens to generate before stopping
		flagFormat         = "format"
		flagClipboard      = "pasteboard"
		flagShowSettings   = "show-config"
		flagMetapromptMode = "metaprompt-mode"
		flagXMLTagContent  = "tag-content"
		flagVariableInput  = "variable-input"
		flagCrossRegion    = "cross-region-inference"
		flagThink          = "think"       // enable thinking for Claude 3.7
		flagBudget         = "budget"      // thinking budget
		flagTextEditor     = "text-editor" // enable text editor tool
		flagImages         = "images"
	)

	rootCmd.PersistentFlags().StringVarP(&config.ModelID, flagModel, string(flagModel[0]), "", "The specific foundation model to use (default is claude-3.5-sonnet)")
	_ = rootCmd.RegisterFlagCompletionFunc(flagModel,
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return AnthrophicModelsIDs, cobra.ShellCompDirectiveDefault
		},
	)
	rootCmd.PersistentFlags().StringVarP(&config.SystemPrompt, flagSystem, "s", "", "The system prompt to use; if given will overwrite template system prompt")
	rootCmd.PersistentFlags().StringVarP(&config.Assistant, flagAssistant, "a", "", "The message for the assistant role")
	rootCmd.PersistentFlags().StringVarP(&config.PromptTemplate, flagPrompt, "p", "", "The prompt name (template) to use")
	_ = rootCmd.RegisterFlagCompletionFunc(flagPrompt,
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			var promptNames []string // fmt.Printf("%v\n", config.Prompts)
			for _, p := range config.Prompts {
				promptNames = append(promptNames, p.Name)
			}
			return promptNames, cobra.ShellCompDirectiveDefault
		},
	)
	rootCmd.PersistentFlags().IntVarP(&config.MaxTokens, flagMaxTokens, string(flagMaxTokens[0]), 0, fmt.Sprintf("The maximum number of tokens to generate before stopping (default=%d)", defaultMaxTokens))
	rootCmd.PersistentFlags().BoolVarP(&config.Format, flagFormat, "f", config.Format, "In prompt ask for the response formatting in markdown unless disabled.")
	rootCmd.PersistentFlags().BoolVarP(&config.Metamode, flagMetapromptMode, "r", config.Metamode, "Treat metaprompt input variable like {$CUSTOMER} like Go templates an interactively ask for input. ")
	rootCmd.PersistentFlags().StringVarP(&config.XMLTagContent, flagXMLTagContent, "x", "", "Write output content within this XML tag name in file <tag name>.txt.")
	rootCmd.PersistentFlags().BoolVarP(&config.ShowSettings, flagShowSettings, "S", false, "Print the bods.yaml settings")
	rootCmd.PersistentFlags().StringVarP(&config.VariableInputRaw, flagVariableInput, "v", "", "Variable input mapping. If provided input will not be asked for interactively. Currently only for metamode e.g. RUBRIC=\"software developer\",RESUME=file://input.txt")
	rootCmd.PersistentFlags().StringVarP(&config.ImagesFlagInput, flagImages, "i", "", "")
	rootCmd.PersistentFlags().BoolVarP(&config.CrossRegionInference, flagCrossRegion, string(flagCrossRegion[0]), config.CrossRegionInference, "Automatically select cross-region inference profile if available for selected model.")

	const darwin = "darwin"
	if runtime.GOOS == darwin {
		rootCmd.PersistentFlags().BoolVarP(&config.Pasteboard, flagClipboard, "P", false, "Get image form pasteboard (clipboard)")
	}

	rootCmd.PersistentFlags().BoolVarP(&config.Think, flagThink, "k", false, "Enable thinking feature for Claude 3.7 model (ignored for other models)")
	rootCmd.PersistentFlags().IntVarP(&config.BudgetTokens, flagBudget, string(flagBudget[0]), 0, fmt.Sprintf("Budget for the max nr of tokens Claude 3.7 may use for thinking (default=%d)", defaultThinkingTokens))
	rootCmd.PersistentFlags().BoolVarP(&config.EnableTextEditor, flagTextEditor, "e", false, "Enable text editor tool for Claude to view and modify files")
}

func main() {
	var err error
	config, err = ensureConfig() // config is global
	if err != nil {
		handleError(bodsError{err, "Could not load configuration file."})
		os.Exit(1)
	}

	// must come after creating the config b/c config values used
	initFlags()

	// With no subcommands, Cobra won't create the default `completion` command.
	// Force creation of the completion related subcommands by adding a 'fake'
	// command when completions are being used. e.g. __BODS_CMP_ENABLED=1 bods completion zsh
	if os.Getenv("__BODS_CMP_ENABLED") == "1" || (len(os.Args) > 1 && os.Args[1] == "__complete") {
		rootCmd.AddCommand(&cobra.Command{Use: "____fake_command_to_enable_completions"})
		rootCmd.InitDefaultCompletionCmd()
	}

	if err := rootCmd.Execute(); err != nil {
		handleError(err)
		os.Exit(1)
	}
}

func handleError(err error) {
	// empty stdin
	if !isInputTerminal() {
		_, _ = io.ReadAll(os.Stdin)
	}

	format := "\n%s\n"

	var args []any
	var bodsErr bodsError
	if errors.As(err, &bodsErr) {
		format += "%s\n\n"
		args = []any{
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorHeader.String(), bodsErr.reason),
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorDetails.Render(err.Error())),
		}
		logger.Println(bodsErr.Error() + " reason: " + bodsErr.reason)
	} else {
		args = []any{
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorDetails.Render(err.Error())),
		}
	}
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func _max100Chars(str string) string {
	if len(str) <= 100 {
		return str
	}
	return str[:100] + "..."
}
