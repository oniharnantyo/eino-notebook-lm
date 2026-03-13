.PHONY: help build run clean test install lint fmt vet

# Application variables
APP_NAME=eino-notebook
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

help: ## Show this help message
	@echo "$(APP_NAME) - Makefile commands"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) .
	@echo "Build complete: bin/$(APP_NAME)"

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME) .
	./bin/$(APP_NAME)

install: ## Install the application to $GOPATH/bin
	@echo "Installing $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(APP_NAME) .
	@echo "Installed to $(GOPATH)/bin/$(APP_NAME)"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it from: https://golangci-lint.run/usage/install/"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe .
	@echo "Build complete"

dev: ## Run in development mode with air (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not found. Install it: go install github.com/cosmtrek/air@latest"; \
	fi

.DEFAULT_GOAL := help
