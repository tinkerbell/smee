#!/usr/bin/env bash

set -eux

failed=0

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
