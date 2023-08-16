all: help

-include lint.mk
-include rules.mk

boots: boots ## Compile boots for host OS and Architecture

crosscompile: $(crossbinaries) ## Compile boots for all architectures

gen: $(generated_go_files) ## Generate go generate'd files

tools: $(toolsBins) ## Builds cli tools defined in tools.go

IMAGE_TAG ?= boots:latest
image: boots-linux-amd64 ## Build docker image
	docker build -t $(IMAGE_TAG) .

test: gen ## Run go test
	CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic -v ${TEST_ARGS} ./...

coverage: test ## Show test coverage
	go tool cover -func=coverage.txt

vet: ## Run go vet
	go vet ./...

goimports: bin/goimports gen ## Run goimports
	goimports -w .

ci-checks: bin/goimports .github/workflows/ci-checks.sh shell.nix gen
	./.github/workflows/ci-checks.sh

ci: ci-checks coverage goimports lint vet ## Runs all the same validations and tests that run in CI

help: ## Print this help
	@grep --no-filename -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed 's/:.*##/·/' | sort | column -ts '·' -c 120
