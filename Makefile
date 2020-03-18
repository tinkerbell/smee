MAKEFLAGS += --no-builtin-rules
.PHONY: ${binary} dc gen test
.SUFFIXES:

binary := boots
all: ${binary}

# this is quick and its really only for rebuilding when dev'ing, I wish go would
# output deps in make syntax like gcc does... oh well this is good enough
${binary}: $(shell git ls-files | grep -v -e vendor -e '_test.go' | grep '.go$$' )
	CGO_ENABLED=0 go build -v -ldflags="-X main.GitRev=$(shell git rev-parse --short HEAD)"

ifeq ($(origin GOBIN), undefined)
GOBIN := ${PWD}/bin
export GOBIN
endif

gen:
	PATH=$$GOBIN:$$PATH go install ./vendor/github.com/jteeuwen/go-bindata/go-bindata
	PATH=$$GOBIN:$$PATH $(MAKE) -C ipxe

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
