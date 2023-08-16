#!/usr/bin/env nix-shell
#!nix-shell -i bash ../../shell.nix
# shellcheck shell=bash

set -eux

failed=0

# --check doesn't show what line number fails, so write the result to disk for the diff to catch
if ! git ls-files '*.json' '*.md' '*.yaml' '*.yml' | xargs prettier --list-different --write; then
	failed=1
fi

if ! shfmt -f . | xargs shfmt -l -d; then
	failed=1
fi

if ! shfmt -f . | xargs shellcheck; then
	failed=1
fi

if ! nixfmt shell.nix; then
	failed=1
fi

if [[ -n $(go run golang.org/x/tools/cmd/goimports@latest -d -e -l .) ]]; then
	go run golang.org/x/tools/cmd/goimports@latest -w .
	failed=1
fi

if ! go mod tidy; then
	failed=true
fi

if ! git diff | (! grep .); then
	failed=1
fi

exit "$failed"
