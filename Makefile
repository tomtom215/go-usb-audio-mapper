BINARY_NAME := usb-soundcard-mapper
MODULE := github.com/tomtom215/go-usb-audio-mapper
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.AppVersion=$(VERSION)
GO := go

.PHONY: all build test test-verbose test-cover lint vet fmt golangci shellcheck e2e clean install help

all: lint shellcheck test e2e build ## Run lint, shellcheck, test, e2e, and build

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

lint: vet fmt golangci ## Run all linters

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Check formatting
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)

golangci: ## Run golangci-lint if installed
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; skipping (see .golangci.yml)"; \
	fi

shellcheck: ## Lint shell scripts and fake-command fixtures (skipped if unavailable)
	@if command -v shellcheck >/dev/null 2>&1; then \
		shellcheck scripts/*.sh testdata/fakebin/*; \
	else \
		echo "shellcheck not installed; skipping"; \
	fi

e2e: ## Run the end-to-end binary smoke test (fake devices, no hardware)
	bash scripts/e2e.sh

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME) coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
