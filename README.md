
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

- include pasteboard (_aka_ clipboard) content in prompt to leverage Claude's vision capability (`-P` includes content of pasteboard in prompt).
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
  -a, --assistant string        The message for the assistant role
  -f, --format                  In prompt ask for the response formatting in markdown unless disabled. (default true)
  -h, --help                    help for bods
  -r, --metaprompt-mode         Treat metaprompt input variable like {$CUSTOMER} like Go templates an interactively ask for input.
  -m, --model string            The specific foundation model to use (default is claude-3-sonnet)
  -P, --pasteboard              Get image form pasteboard (clipboard)
  -p, --prompt string           The prompt name (template) to use
  -S, --show-config             Print the bods.yaml settings
  -s, --system string           The system prompt to use; if given will overwrite template system prompt
  -x, --tag-content string      Write output content within this XML tag name in file <tag name>.txt.
  -t, --tokens int              The maximum number of tokens to generate before stopping (default=400)
  -v, --variable-input string   Variable input mapping. If provided input will not be asked for interactively. Currently only for metamode e.g. RUBRIC="software developer",RESUME=file://input.txt
      --version                 version for bods
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

### Built-in Metaprompt

One of the included prompts (in 'bods.yaml') is Anthropic's “meta”-prompt.

As per Antrhopic:
> The Metaprompt is a long multi-shot prompt filled with half a dozen examples
> of good prompts for solving various tasks. These examples help Claude to write
> a good prompt for your task.

See [Helper metaprompt(experimental)](https://docs.anthropic.com/en/docs/helper-metaprompt-experimental)
for details.

To use the metaprompt the recommended command is
```sh
bods --prompt metaprompt` --format=false --tag-content Instructions -t 4096
```

`bods` will then interactively ask you to enter the the task.

`--format=false` The metaprompt output includes many XML tag, these might not be
rendered correctly (or 'swallowed') in rendered markdown. To see the full raw
output, disabling markdown formatting and rendering is recommended.

`--tag-content=Instructions` Will write any output within the
```xml
`<Instructions></Instructions>`
```
tags in a file with the name of the tag - in this case `Instructions.txt`. The
way the metaprompt is structured, it outputs
the generted prompt within `<Instructions>` xml tags.

The generated prompt will have so called 'input variables'. For example, a
task like "Draft an email responding to a customer complaint" an input
variable will likely be `CUSTOMER_COMPLAINT`; snippet of the generated prompt

```xml
<Instructions>
You will be drafting a professional email response to a customer complaint.

Here is the text of the customer complaint:
<customer_complaint>{$CUSTOMER_COMPLAINT}</customer_complaint>
...
```

When using the generate prompt, you will need to 1/ instruct `bods` to 'watch
out' for these input variables that are part of the prompt generated by
metaprompt 2/ supply values for these inputs

1/ is addressed by passing the parameter `--metaprompt-mode` to `bods`

2/ here you have two options:

__2a__ if you just pass `--metaprompt-mode` and don't to anything else, `bods`
will ask you for the required input varibles interactively

__2b__ you can pass the the values for these input variable as part of the
parameters e.g.

```sh
cat Instructions.txt | bods --metaprompt-mode --variable-input CUSTOMER_COMPLAINT="It's been four weeks and I still haven't received my order"
```

or to use file content as a value for an input value:

```sh
cat Instructions.txt | bods --metaprompt-mode --variable-input CUSTOMER_COMPLAINT=file://complaint_email.txt
```

A full workflow for using the metaprompt to generate a prompt for the task

```
Review a AWS Cloudformation yaml template against architectural and security best practices"
```

and then using the generated prompt for reviewing an existing Cloudformation
file is shown here:

![metaprompt](https://github.com/rollwagen/bods/assets/7364201/c34bc835-6c92-4744-ae56-58f86753d9b7)


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
