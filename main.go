package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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
		f, err := tea.LogToFile(path, "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		// defer f.Close()

		_, _ = fmt.Fprintf(os.Stderr, "DEBUG logging to file %s\n", path)

		logger = log.New(f, "debug ", log.Lmsgprefix)

		logger.Println(time.Now().Format("2006-01-02 15:04:05.000000") + " starting mods...")
	} else {
		logger = log.New(io.Discard, "", 0)
	}

	glamour.DarkStyleConfig.CodeBlock.Chroma.Error.BackgroundColor = new(string)
	glamour.LightStyleConfig.CodeBlock.Chroma.Error.BackgroundColor = new(string)

	buildVersion()

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

var (
	config Config

	logger *log.Logger

	rootCmd = &cobra.Command{
		Use:           "bods",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			config.Prefix = strings.Join(args, " ")
			logger.Println("config.Prefix: " + config.Prefix)

			opts := []tea.ProgramOption{
				tea.WithOutput(stderrRenderer().Output()),
			}

			if !isInputTerminal() {
				opts = append(opts, tea.WithInput(nil)) // To disable input entirely pass nil
			}

			bods := initialBodsModel(stderrRenderer(), &config)
			logger.Println("starting new tea program...")
			p := tea.NewProgram(bods, opts...)
			m, err := p.Run()
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
		flagModel        = "model"  // the specific model to use
		flagSystem       = "system" // system prompt
		flagPrompt       = "prompt" // prompt name (template) to use
		flagMaxTokens    = "tokens" // max nr of tokens to generate before stopping
		flagFormat       = "format"
		flagClipboard    = "pasteboard"
		flagShowSettings = "show-config"
	)

	rootCmd.PersistentFlags().StringVarP(&config.ModelID, flagModel, string(flagModel[0]), "", "The specific foundation model to use")
	_ = rootCmd.RegisterFlagCompletionFunc(flagModel,
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return AnthrophicModelsIDs, cobra.ShellCompDirectiveDefault
		},
	)
	rootCmd.PersistentFlags().StringVarP(&config.SystemPrompt, flagSystem, "s", "", "The system prompt to use; if given will overwrite template system prompt")
	rootCmd.PersistentFlags().StringVarP(&config.PromptTemplate, flagPrompt, "p", "", "The prompt name (template) to use")
	_ = rootCmd.RegisterFlagCompletionFunc(flagPrompt,
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			var promptNames []string
			// fmt.Printf("%v\n", config.Prompts)
			for _, p := range config.Prompts {
				promptNames = append(promptNames, p.Name)
			}
			return promptNames, cobra.ShellCompDirectiveDefault
		},
	)
	rootCmd.PersistentFlags().IntVarP(&config.MaxTokens, flagMaxTokens, string(flagMaxTokens[0]), 0, "The maximum number of tokens to generate before stopping")
	rootCmd.PersistentFlags().BoolVarP(&config.Format, flagFormat, "f", config.Format, "In prompt ask for the response formatting in markdown unless disabled.")

	// rootCmd.PersistentFlags().BoolVarP(&config.Format, flagFormat, "f", config.Format, "In prompt ask for the response formatting in markdown unless disabled.")
	rootCmd.PersistentFlags().BoolVarP(&config.ShowSettings, flagShowSettings, "S", false, "Print the embedded bods.yaml settings")

	const darwin = "darwin"
	if runtime.GOOS == darwin {
		rootCmd.PersistentFlags().BoolVarP(&config.Pasteboard, flagClipboard, "P", false, "Get image form pasteboard (clipboard)")
	}
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

	var args []interface{}
	var bodsErr bodsError
	if errors.As(err, &bodsErr) {
		format += "%s\n\n"
		args = []interface{}{
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorHeader.String(), bodsErr.reason),
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorDetails.Render(err.Error())),
		}
		logger.Println(bodsErr.Error() + " reason: " + bodsErr.reason)
	} else {
		args = []interface{}{
			stderrStyles().ErrPadding.Render(stderrStyles().ErrorDetails.Render(err.Error())),
		}
	}
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}
