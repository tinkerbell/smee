all: help

-include rules.mk

boots: cmd/boots/boots ## Compile boots for host OS and Architecture

crosscompile: $(crossbinaries) ## Compile boots for all architectures

gen: $(generated_go_files) ## Generate go generate'd files

tools: $(toolsBins) ## Builds cli tools defined in tools.go

IMAGE_TAG ?= boots:latest
image: cmd/boots/boots-linux-amd64 ## Build docker image
	docker build -t $(IMAGE_TAG) .

test: gen setup-envtest## Run go test
	KUBEBUILDER_ASSETS=$(ASSETS) CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic ${TEST_ARGS} ./...

coverage: test ## Show test coverage
	go tool cover -func=coverage.txt

vet: ## Run go vet
	go vet ./...

goimports: bin/goimports gen ## Run goimports
	goimports -w .

golangci-lint: bin/golangci-lint gen ## Run golangci-lint
	golangci-lint run -v --timeout=5m

ci-checks: bin/goimports .github/workflows/ci-checks.sh shell.nix gen
	./.github/workflows/ci-checks.sh

ci: ci-checks coverage goimports golangci-lint vet ## Runs all the same validations and tests that run in CI

help: ## Print this help
	@grep --no-filename -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed 's/:.*##/·/' | sort | column -ts '·' -c 120

setup-envtest:
ifeq (, $(shell which /usr/local/kubebuilder/bin/kube-apiserver))
	@{ \
	set -e ;\
	mkdir -p /tmp/kubebuilder ;\
	K8S_VERSION="1.21.2" ;\
	GOOS=$$(go env GOOS) ;\
	GOARCH=$$(go env GOARCH) ;\
    curl -Lo envtest-bins.tar.gz "https://go.kubebuilder.io/test-tools/$${K8S_VERSION}/$${GOOS}/$${GOARCH}" ;\
    tar -C /tmp/kubebuilder  --strip-components=1 -zvxf envtest-bins.tar.gz ;\
    rm envtest-bins.tar.gz ;\
	}
export ASSETS=/tmp/kubebuilder/bin
else
export ASSETS=/usr/local/kubebuilder/bin/
endif