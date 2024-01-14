#!/bin/sh
CLI_NAME="bods"
set -e
rm -rf completions
mkdir completions
for sh in "bash" "zsh" "fish"; do
	__BODS_CMP_ENABLED=1 go run . completion "${sh}" > "completions/${CLI_NAME}.${sh}"
done
