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

.PHONY: all boots crosscompile dc image gen run test

CGO_ENABLED := 0
export CGO_ENABLED

GitRev := $(shell git rev-parse --short HEAD)
SOURCE_DATE_EPOCH := $(shell git log -1 --pretty=%ct)
export SOURCE_DATE_EPOCH

crossbinaries := boots-linux-amd64 boots-linux-arm64
boots-linux-amd64: FLAGS=GOARCH=amd64
boots-linux-arm64: FLAGS=GOARCH=arm64
boots-linux-amd64 boots-linux-arm64: boots
	${FLAGS} GOOS=linux go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ .

ifeq ($(origin GOBIN), undefined)
GOBIN := ${PWD}/bin
export GOBIN
PATH := ${GOBIN}:${PATH}
export PATH
endif

# parses tools.go and returns the tool name prefixed with bin/
toolsBins := $(addprefix bin/,$(notdir $(shell grep '^\s*_' tools.go | awk -F'"' '{print $$2}')))

# build cli tools defined in tools.go
$(toolsBins): go.mod go.sum tools.go
$(toolsBins): CMD=$(shell awk -F'"' '/$(@F)"/ {print $$2}' tools.go)
$(toolsBins):
	go install "$(CMD)"

generated_go_files := \
	syslog/facility_string.go \
	syslog/severity_string.go \

# go generate
go_generate: $(generated_go_files)
$(filter %_string.go,$(generated_go_files)): bin/stringer
syslog/facility_string.go: syslog/message.go
syslog/severity_string.go: syslog/message.go
$(generated_go_files): bin/goimports
	go generate -run="$(@F)" ./...
	goimports -w $@

boots: syslog/facility_string.go syslog/severity_string.go cleanup
	go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ .

cleanup:
	rm -f boots boots-linux-amd64 boots-linux-arm64