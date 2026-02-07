[![GitHub Release](https://img.shields.io/github/release/rollwagen/bods.svg)](https://github.com/rollwagen/bods/releases)
![Downloads](https://img.shields.io/github/downloads/rollwagen/bods/total)
[![Go Reference](https://pkg.go.dev/badge/github.com/rollwagen/bods.svg)](https://pkg.go.dev/github.com/rollwagen/bods)
[![Go Report Card](https://goreportcard.com/badge/github.com/rollwagen/bods)](https://goreportcard.com/report/github.com/rollwagen/bods)
![CodeQL](https://github.com/rollwagen/bods/workflows/CodeQL/badge.svg)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

<div align="center">
<img src="https://github.com/rollwagen/bods/assets/7364201/2c1d4116-6457-41ab-856b-254e6aa6d661"/>
</div>


# `bods`

Like [mods](https://github.com/charmbracelet/mods), but for [Anthropic's Claude
models on Amazon **B**edrock](https://aws.amazon.com/bedrock/claude/)

[Claude](https://www.anthropic.com/claude) for the command line, with support
for Unix like piping (`|`) and file redirecting (`<`).

- **PDF Support**: Pipe PDFs directly or read from pasteboard. `bods` extracts text and sends the PDF as a document.
- **Thinking / Reasoning**: Support for thinking capabilities (`-k` or `--think`) for Claude 3.7 and later models. For Opus 4.6, use `--effort` to control adaptive thinking.
- **Text Editor Tool**: Allow Claude to view and modify files directly (`-e` or `--text-editor`).
- **Images & Pasteboard**: Include pasteboard content (images, text, PDFs) in prompt (`-P`).
- **Autocomplete**: Enabled for flags, params, and prompts (hit `<TAB><TAB>`).
- **Pre-configured Prompts**: See [bods.yaml](https://github.com/rollwagen/bods/blob/main/bods.yaml).
- **Supported Models**:
    - Claude Opus 4.6 (default)
    - Claude 4.5 (Sonnet, Haiku, Opus)
    - Claude 3.7 Sonnet
    - Claude 3.5 (Sonnet, Haiku)
    - Legacy models: Claude 3 (Opus, Sonnet, Haiku), Claude 2.x

## Usage

```sh
$ bods --help
Usage:
  bods [flags]

Flags:
  -a, --assistant string         The message for the assistant role
  -b, --budget int               Thinking token budget for Claude 3.7-4.5; ignored for Opus 4.6, use --effort instead (default=1024)
  -c, --cross-region-inference   Automatically select cross-region inference profile if available for selected model. (default true)
  -E, --effort string            Effort level (max, high, medium, low). 'max' is Opus 4.6 only.
  -f, --format                   In prompt ask for the response formatting in markdown unless disabled. (default true)
  -h, --help                     help for bods
  -i, --images string
  -r, --metaprompt-mode          Treat metaprompt input variable like {$CUSTOMER} like Go templates an interactively ask for input.
  -m, --model string             The specific foundation model to use (default is claude-opus-4.6)
  -P, --pasteboard               Get image form pasteboard (clipboard)
  -p, --prompt string            The prompt name (template) to use
  -S, --show-config              Print the bods.yaml settings
  -s, --system string            The system prompt to use; if given will overwrite template system prompt
  -x, --tag-content string       Write output content within this XML tag name in file <tag name>.txt.
  -e, --text-editor              Enable text editor tool for Claude to view and modify files
  -k, --think                    Enable thinking (extended for 3.7-4.5, adaptive for Opus 4.6)
  -t, --tokens int               The maximum number of tokens to generate before stopping (default=2048)
  -v, --variable-input string    Variable input mapping. If provided input will not be asked for interactively.
      --version                  version for bods
```

## Install Bods

### Pre-requisites

1. An AWS account and credentials specified (e.g. via environment variables `AWS_ACCESS_KEY_ID,` `AWS_SECRET_ACCESS_KEY,` `AWS_SESSION_TOKEN`, `AWS_REGION`); for details see [AWS SDK for Go V2 - Specifying Credentials](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials)
2. You need access to Anthropic's Claude models granted in Bedrock, see [Amazon Bedrock - Model access](https://docs.aws.amazon.com/bedrock/latest/userguide/model-access.html)

### Installation

```sh
brew install rollwagen/tap/bods
```
_OR_

```sh
go install github.com/rollwagen/bods@latest
```

_OR_

Download
[releases]: https://github.com/rollwagen/bods/releases

### Enable CLI completion

* when installed with `brew`, autocompletion is installed and enabled for zsh and bash per default.
* to enable manually (e.g. during development):
ZSH example:

```sh
__BODS_CMP_ENABLED=1 bods completion zsh > b_cmp.sh; source b_cmp.sh; rm b_cmp.sh
```

## Features & Examples

### Zero-Shot

```sh
$ bods "Print a CLI shell command that uses curl to check the cluster health of an OpenSearch endpoint listening on port 9200"

curl -XGET 'http://localhost:9200/_cluster/health?pretty'
```

### Thinking / Reasoning (Claude 3.7+)

Enable extended thinking capabilities for supported models (Claude 3.7 and later) to solve complex problems.

```sh
# Extended thinking for Claude 3.7-4.5
$ bods "Explain the solution to the Riemann Hypothesis" -k --budget 4000

# Adaptive thinking for Opus 4.6 via effort level
$ bods "Explain the solution to the Riemann Hypothesis" --effort high
```

### Text Editor Tool

Allow Claude to view and modify files in your current directory. Useful for refactoring or fixing bugs.

```sh
$ bods "Fix the syntax error in main.go" -e
```

### PDF Support

Pipe PDFs directly into `bods` or use the pasteboard flag `-P` if you have a PDF copied.

```sh
# Pipe a PDF
$ cat paper.pdf | bods "Summarize the key findings"

# Use PDF from clipboard
$ bods "What is this document about?" -P
```

### Piping & Multimodal

Summarize a YouTube video: get a YouTube transcript with [ytt](https://github.com/rollwagen/hacks/tree/main/youtube-transcript) and pipe to `bods`.

```sh
ytt "https://youtube.com/..." | bods -p summarize
```

Explain what specific source code does.

![bods_code](https://github.com/rollwagen/bods/assets/7364201/5ffb3de5-372f-44fa-982a-f211136fa581)

### Built-in Metaprompt

Use Anthropic's [Metaprompt](https://docs.anthropic.com/en/docs/helper-metaprompt-experimental) to generate high-quality prompts.

```sh
bods --prompt metaprompt --format=false --tag-content Instructions -t 4096
```

`bods` will interactively ask you to enter the task.

For input variables in the generated prompt:

```sh
cat Instructions.txt | bods --metaprompt-mode --variable-input CUSTOMER_COMPLAINT=file://complaint_email.txt
```

![metaprompt](https://github.com/rollwagen/bods/assets/7364201/c34bc835-6c92-4744-ae56-58f86753d9b7)

## Configuration (`bods.yaml`)

Define your own prompts and defaults in `~/.config/bods/bods.yaml` (or macOS: `~/Library/Application Support/bods/bods.yaml`).

```yaml
prompts:
  demo:
    max_tokens: 1000
    user: |
      USER PROMPT TEXT from 'demo'
    system: |
      SYSTEM PROMPT TEXT from 'demo'
```

## Debugging

### Dump constructed prompt

```sh
$ DUMP_PROMPT=1 bods "hello"
```

### Debug log

```sh
$ DEBUG=1 bods "hello"
# Log written to temp directory (e.g. /tmp/bods.log)
```

## License

[MIT](https://github.com/rollwagen/bods/raw/main/LICENSE)

## Acknowledgments

Inspiration
- [mods](https://github.com/charmbracelet/mods) by [charm.sh](https://charm.sh/)

Similar projects
- [aichat](https://github.com/sigoden/aichat)
- [fabric](https://github.com/danielmiessler/fabric)
