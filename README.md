# Bods!

[![GitHub Release](https://img.shields.io/github/release/rollwagen/bods.svg)](https://github.com/rollwagen/bods/releases)
![Downloads](https://img.shields.io/github/downloads/rollwagen/bods/total)
[![Go Reference](https://pkg.go.dev/badge/github.com/rollwagen/bods.svg)](https://pkg.go.dev/github.com/rollwagen/bods)
[![Go Report Card](https://goreportcard.com/badge/github.com/rollwagen/bods)](https://goreportcard.com/report/github.com/rollwagen/bods)
![CodeQL](https://github.com/rollwagen/bods/workflows/CodeQL/badge.svg)
[![Powered By: GoReleaser](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=flat-square)](https://github.com/goreleaser)

![1405a00cffea95906c4a7343790db2dc08821b64](https://github.com/rollwagen/bods/assets/7364201/2c1d4116-6457-41ab-856b-254e6aa6d661)

Like [mods](https://github.com/charmbracelet/mods), but for **B**edrock.

## Examples

**Zero-Shot**

```sh
$ bods "Print a CLI shell command that uses curl to check the cluster health of an OpenSearch endpoint listening on port 9200"

curl -XGET 'http://localhost:9200/_cluster/health?pretty'

```

**Piping**

Summarize a YouTube video: get a YouTube transcript with [ytt](https://github.com/rollwagen/hacks/tree/main/youtube-transcript) and pipe to `bods` using a prompt-template for summarization.

Video URL: "AWS re:Invent 2023 - Prompt engineering best practices for LLMs on Amazon Bedrock (AIM377)"

![bods_ytt](https://github.com/rollwagen/bods/assets/7364201/cff9bb2e-aee0-4119-ac55-96eddd1d85dc)


Explain what specific source code does.

![bods_code](https://github.com/rollwagen/bods/assets/7364201/5ffb3de5-372f-44fa-982a-f211136fa581)


## Prompt construction

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


## Install Bods

```sh
brew install rollwagen/tap/bods
```
_OR_

```sh
go install github.com/rollwagen/bods@latest
```
### Enable CLI complestion

When installing with `brew`, auto completion is installed and enabled for zsh and bash.

ZSH example:

```sh
__BODS_CMP_ENABLED=1 bods completion zsh > b_cmp.sh; source b_cmp.sh; rm b_cmp.sh
```

## License

[MIT](https://github.com/rollwagen/bods/raw/main/LICENSE)

---
