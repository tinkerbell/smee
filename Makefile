all: help

-include lint.mk
-include rules.mk

build: cmd/smee/smee ## Compile smee for host OS and Architecture

crosscompile: $(crossbinaries) ## Compile smee for all architectures

gen: $(generated_go_files) ## Generate go generate'd files

IMAGE_TAG ?= smee:latest
image: cmd/smee/smee-linux-amd64  ## Build docker image
	docker build -t $(IMAGE_TAG) .

test: gen ## Run go test
	CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic -v ${TEST_ARGS} ./...

coverage: test ## Show test coverage
	go tool cover -func=coverage.txt

vet: ## Run go vet
	go vet ./...

goimports: gen ## Run goimports
	$(GOIMPORTS) -w .

ci-checks: .github/workflows/ci-checks.sh gen
	./.github/workflows/ci-checks.sh

ci: ci-checks coverage goimports lint vet ## Runs all the same validations and tests that run in CI

help: ## Print this help
	@grep --no-filename -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed 's/:.*##/·/' | sort | column -ts '·' -c 120
