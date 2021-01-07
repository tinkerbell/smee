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

binary := boots
.PHONY: all ${binary} crosscompile dc gen run test
all: ${binary}

crosscompile: $(shell git ls-files | grep -v -e vendor -e '_test.go' | grep '.go$$' )
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -v -o ./boots-linux-x86_64 -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./boots-linux-amd64 -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build -v -o ./boots-linux-aarch64 -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -v -o ./boots-linux-armv7l -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o ./boots-linux-arm64 -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"


# this is quick and its really only for rebuilding when dev'ing, I wish go would
# output deps in make syntax like gcc does... oh well this is good enough
${binary}: $(shell git ls-files | grep -v -e vendor -e '_test.go' | grep '.go$$' )
	CGO_ENABLED=0 go build -v -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"

ifeq ($(origin GOBIN), undefined)
GOBIN := ${PWD}/bin
export GOBIN
endif

ipxe/bindata.go:
	$(MAKE) -C ipxe

ifeq ($(CI),drone)
run: ${binary}
	${binary}
test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ${TEST_ARGS} ./...
else
run: ${binary}
	docker-compose up -d --build cacher
	docker-compose up --build boots
test:
	docker-compose up -d --build cacher
endif
