all: help

-include rules.mk

boots: cmd/boots/boots ## Compile boots for host OS and Architecture

bindata: ipxe/bindata.go ## Build and generate embedded iPXE binaries

crosscompile: $(crossbinaries) ## Compile boots for all architectures
	
gen: $(generated_files) ## Generate go generate'd files

image: cmd/boots/boots-linux-amd64 ## Build docker image
	docker build -t boots .

test: gen ## Run go test
	CGO_ENABLED=1 go test -race -coverprofile=coverage.txt -covermode=atomic -gcflags=-l ${TEST_ARGS} ./...

coverage: test ## Show test coverage
	go tool cover -func=coverage.txt

vet: ## Run go vet
	go vet ./...

goimports: ## Run goimports
	@echo be sure goimports is installed
	goimports -w .

golangci-lint: ## Run golangci-lint
	@echo be sure golangci-lint is installed: https://golangci-lint.run/usage/install/
	golangci-lint run

validate-local: vet coverage goimports golangci-lint ## Runs all the same validations and tests that run in CI

help: ## Print this help
	@grep --no-filename -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sed 's/:.*##/·/' | sort | column -ts '·' -c 120
