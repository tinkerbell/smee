# Only use the recipes defined in these makefiles
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:
# Delete target files if there's an error
# This avoids a failure to then skip building on next run if the output is created by shell redirection for example
# Not really necessary for now, but just good to have already if it becomes necessary later.
.DELETE_ON_ERROR:
# Treat the whole recipe as a one shell script/invocation instead of one-per-line
.ONESHELL:
# Use bash instead of plain sh
SHELL := bash
.SHELLFLAGS := -o pipefail -euc

# Runnable tools
GO			?= go
GOIMPORTS	:= $(GO) run golang.org/x/tools/cmd/goimports@latest

.PHONY: all smee crosscompile dc image gen run test

CGO_ENABLED := 0
export CGO_ENABLED

GitRev := $(shell git rev-parse --short HEAD)

crossbinaries := cmd/smee/smee-linux-amd64 cmd/smee/smee-linux-arm64
cmd/smee/smee-linux-amd64: FLAGS=GOARCH=amd64
cmd/smee/smee-linux-arm64: FLAGS=GOARCH=arm64
cmd/smee/smee-linux-amd64 cmd/smee/smee-linux-arm64: smee
	${FLAGS} GOOS=linux go build -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/smee/

generated_go_files := \
	internal/syslog/facility_string.go \
	internal/syslog/severity_string.go \

# go generate
go_generate: $(generated_go_files)
$(filter %_string.go,$(generated_go_files)):
internal/syslog/facility_string.go: internal/syslog/message.go
internal/syslog/severity_string.go: internal/syslog/message.go
$(generated_go_files):
	go generate -run="$(@F)" ./...
	$(GOIMPORTS) -w $@

cmd/smee/smee: internal/syslog/facility_string.go internal/syslog/severity_string.go cleanup
	go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/smee/

cleanup:
	rm -f cmd/smee/smee cmd/smee/smee-linux-amd64 cmd/smee/smee-linux-arm64