
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

- include pasteboard (_aka_ clipboard) content in prompt to leverage Claude's vision capability (`-P` inludes content of pasteboard in prompt).
- autocomplete for flags, params etc (hit `<TAB><TAB>`)
- pre-configures prompts (autocomplete enabled), see [bods.yaml](https://github.com/rollwagen/bods/blob/main/bods.yaml)
- supported Claude models: "anthropic.claude-v2", "anthropic.claude-v2:1", "anthropic.claude-3-sonnet-20240229-v1:0", "anthropic.claude-3-haiku-20240307-v1:0", "anthropic.claude-3-opus-20240229-v1:0"
- set [system prompt](https://docs.anthropic.com/claude/docs/system-prompts) directly via `--system` or in config file 'bods.yaml'

## Usage

```sh
$ bods --help

Usage:
  bods [flags]

Flags:
  -f, --format          In prompt ask for the response formatting in markdown unless disabled. (default true)
  -h, --help            help for bods
  -m, --model string    The specific foundation model to use
  -P, --pasteboard      Get image form pasteboard (clipboard)
  -p, --prompt string   The prompt name (template) to use
  -S, --show-config     Print the embedded bods.yaml settings
  -s, --system string   The system prompt to use; if given will overwrite template system prompt
  -t, --tokens int      The maximum number of tokens to generate before stopping
  -v, --version         version for bods
```

## Install Bods

### Pre-requisites

1. An AWS account and credentials specified (e.g. via environment variables `AWS_ACCESS_KEY_ID,` `AWS_SECRET_ACCESS_KEY,` `AWS_SESSION_TOKEN`, `AWS_REGION`); for details see [AWS SDK for Go V2 - Specifying Credentials](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials)
2. Access to Anthropic's Claude models granted in Bedrock, see [Amazon Bedrock - Model access](https://docs.aws.amazon.com/bedrock/latest/userguide/model-access.html)

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

### Enable CLI complestion

* when installecd with `brew`, autocompletion is installed and enabled for zsh and bash per default.
* to enable manually (e.g. during development):
ZSH example:

```sh
__BODS_CMP_ENABLED=1 bods completion zsh > b_cmp.sh; source b_cmp.sh; rm b_cmp.sh
```

## Examples

### Zero-Shot

```sh
$ bods "Print a CLI shell command that uses curl to check the cluster health of an OpenSearch endpoint listening on port 9200"

curl -XGET 'http://localhost:9200/_cluster/health?pretty'

```

### Piping

Summarize a YouTube video: get a YouTube transcript with [ytt](https://github.com/rollwagen/hacks/tree/main/youtube-transcript) and pipe to `bods` using a prompt-template for summarization.

Video URL: "AWS re:Invent 2023 - Prompt engineering best practices for LLMs on Amazon Bedrock (AIM377)"

![bods_ytt](https://github.com/rollwagen/bods/assets/7364201/cff9bb2e-aee0-4119-ac55-96eddd1d85dc)


Explain what specific source code does.

![bods_code](https://github.com/rollwagen/bods/assets/7364201/5ffb3de5-372f-44fa-982a-f211136fa581)


## Prompt construction for various piped and passed inputs

```sh
$ echo "PIPED INPUT" | bods --prompt demo "ARGUMENT" < file.txt
```

```json
{
  "anthropic_version": "bedrock-2023-05-31",
  "messages": [
    {
      "content": "\n\nUSER PROMPT TEXT from 'demo'\n ARGUMENT\n\nPIPED INPUT\nFILE CONTENT\n\n\n Format the response as markdown without enclosing backticks.\n\n",
      "role": "user"
    }
  ],
  "system": "SYSTEM PROMPT TEXT from 'demo'\n",
  "temperature": 1,
  "max_tokens": 1000,
  "top_p": 0.999
}
```

`bods.yaml` 'demo' prompt config:

```yaml
demo:
 max_tokens: 1000
 user: |
   USER PROMPT TEXT from 'demo'
 system: |
  SYSTEM PROMPT TEXT from 'demo'
```

## Debugging

### Dump contructed prompt and exit

```sh
$ DUMP_PROMPT=1 bods "hello"
```

```json
{
  "anthropic_version": "bedrock-2023-05-31",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "\n hello \n\n Format the response as markdown without enclosing backticks.\n"
        }
      ]
    }
  ],
  "temperature": 1,
  "max_tokens": 200,
  "top_p": 0.999
}
```

### Debug log

⚠️  The debug log will contain the complete prompt, including all input passed and included (e.g. via file redirect).

```sh
$ DEBUG=1 bods "hello"
DEBUG logging to file /var/folders/jr/43nrkrhj4dg25x1vb035m1n00000gn/T/bods.log
```

```sh
$ cat /var/folders/jr/43nrkrhj4dg25x1vb035m1n00000gn/T/bods.log

debug 2024-04-01 18:20:22.617622 starting mods...
debug config directories: [/Users/rollwagen/Library/Preferences ~/Application Support /Library/Preferences]
debug replacing standard embedded bods.yaml with file content form ~/Library/Application Support/bods/bods.yaml
debug adding prompt from config: summarize
debug config.Prefix: hello
debug starting new tea program...
debug current state is 'startState', appending 'readStdinCmd'
debug readStdInCmd: isInputTerminal=true
debug startMessagesCmd: len(content)=0
debug config.ModelID set to:  anthropic.claude-3-sonnet-20240229-v1:0
debug config.PromptTemplate=  config.SystemPrompt=
...
```


## License

[MIT](https://github.com/rollwagen/bods/raw/main/LICENSE)

## Acknowledgments

Inspiration
- Quite obviously: [mods](https://github.com/charmbracelet/mods) by [charm.sh](https://charm.sh/)

Similar projects
- [aichat](https://github.com/sigoden/aichat)
- [fabric](https://github.com/danielmiessler/fabric) by [D. Miessler](https://danielmiessler.com/)


---
