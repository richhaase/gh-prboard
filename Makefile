.PHONY: help build test test-race test-coverage fmt lint vet tidy clean check

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build gh-prboard binary
	go build -ldflags "$(LDFLAGS)" -o bin/gh-prboard .

test: ## Run tests with verbose output
	go test ./... -v

test-race: ## Run tests with race detector
	go test -race ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

fmt: ## Check formatting (fail if files need formatting)
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

lint: ## Run golangci-lint
	golangci-lint run

vet: ## Run go vet
	go vet ./...

tidy: ## Run go mod tidy
	go mod tidy

clean: ## Remove build artifacts and test cache
	rm -rf bin/
	go clean -testcache

check: fmt vet lint test ## Run fmt, vet, lint, and test
