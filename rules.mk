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

crossbinaries := cmd/boots/boots-linux-386 cmd/boots/boots-linux-amd64 cmd/boots/boots-linux-arm64 cmd/boots/boots-linux-armv6 cmd/boots/boots-linux-armv7
cmd/boots/boots-linux-386:   FLAGS=GOARCH=386
cmd/boots/boots-linux-amd64: FLAGS=GOARCH=amd64
cmd/boots/boots-linux-arm64: FLAGS=GOARCH=arm64
cmd/boots/boots-linux-armv6: FLAGS=GOARCH=arm GOARM=6
cmd/boots/boots-linux-armv7: FLAGS=GOARCH=arm GOARM=7
cmd/boots/boots-linux-386 cmd/boots/boots-linux-amd64 cmd/boots/boots-linux-arm64 cmd/boots/boots-linux-armv6 cmd/boots/boots-linux-armv7: boots
	${FLAGS} GOOS=linux go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/boots/

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
	client/cacher/mock_cacher/cacher_mock.go \
	client/tinkerbell/mock_workflow/workflow_mock.go \
	client/tinkerbell/mock_hardware/hardware_mock.go \
	syslog/facility_string.go \
	syslog/severity_string.go \

# go generate
go_generate: $(generated_go_files)
$(filter %_string.go,$(generated_go_files)): bin/stringer
$(filter %_mock.go,$(generated_go_files)): bin/mockgen
client/cacher/mock_cacher/cacher_mock.go: client/cacher/discovery.go
client/tinkerbell/mock_workflow/workflow_mock.go: client/tinkerbell/discovery.go
client/tinkerbell/mock_hardware/hardware_mock.go: client/tinkerbell/discovery.go
syslog/facility_string.go: syslog/message.go
syslog/severity_string.go: syslog/message.go
$(generated_go_files): bin/goimports
	go generate -run="$(@F)" ./...
	goimports -w $@

# this is quick and its really only for rebuilding when dev'ing, I wish go would
# output deps in make syntax like gcc does... oh well this is good enough
cmd/boots/boots: $(shell git ls-files | grep -v -e vendor -e '_test.go' -e 'mock' | grep '.go$$' ) syslog/facility_string.go syslog/severity_string.go
	go build -v -ldflags="-X main.GitRev=${GitRev}" -o $@ ./cmd/boots/
