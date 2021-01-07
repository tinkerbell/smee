MAKEFLAGS += --no-builtin-rules
.SUFFIXES:

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
