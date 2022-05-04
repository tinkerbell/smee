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

if ! git ls-files '*.go' | xargs -I% sh -c 'sed "/^import (/,/^)/ { /^\s*$/ d }" % >%.tmp && goimports -w %.tmp && (if cmp -s % %.tmp; then rm %.tmp; else mv %.tmp %; fi)'; then
	failed=1
fi

if ! go mod tidy; then
	failed=true
fi

if ! git diff | (! grep .); then
	failed=1
fi

exit "$failed"
