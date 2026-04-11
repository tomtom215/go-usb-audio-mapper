BINARY_NAME := usb-soundcard-mapper
MODULE := github.com/tomtom215/go-usb-audio-mapper
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.AppVersion=$(VERSION)
GO := go

.PHONY: all build test test-verbose test-cover lint vet fmt clean install help

all: lint test build ## Run lint, test, and build

build: ## Build the binary
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

install: build ## Install the binary to /usr/local/bin
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

test: ## Run tests
	$(GO) test -count=1 -race ./...

test-verbose: ## Run tests with verbose output
	$(GO) test -v -count=1 -race ./...

test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out -race ./...
	$(GO) tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report: go tool cover -html=coverage.out"

lint: vet fmt ## Run all linters

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Check formatting
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME) coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
