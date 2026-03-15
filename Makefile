.PHONY: help build run clean test install lint fmt vet migrate-install migrate-up migrate-down migrate-steps migrate-create migrate-force

# Application variables
APP_NAME=eino-notebook
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Database variables (from .env or defaults)
DATABASE_HOST?=localhost
DATABASE_PORT?=5432
DATABASE_USER?=postgres
DATABASE_PASSWORD?=password
DATABASE_NAME?=eino_notebook
DATABASE_URL?=postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=disable

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
		echo "air not found. Install it: go install github.com/air-verse/air@latest"; \
	fi

# Migration targets
migrate-install: ## Install go-migrate CLI
	@echo "Installing go-migrate..."
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "migrate installed to ~/go/bin/migrate"

migrate-up: ## Run all up migrations
	@echo "Running migrations up..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" up; \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

migrate-down: ## Run all down migrations
	@echo "Running migrations down..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" down; \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

migrate-steps: ## Run specific number of migrations (use STEPS=n)
	@echo "Running $(STEPS) migration steps..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" up $(STEPS); \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

migrate-create: ## Create a new migration (use NAME=migration_name)
	@echo "Creating migration: $(NAME)..."
	@if command -v migrate > /dev/null; then \
		migrate create -ext sql -dir migrations -seq $(NAME); \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

migrate-force: ## Force migration version (use VERSION=n)
	@echo "Forcing migration version to $(VERSION)..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" force $(VERSION); \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

migrate-version: ## Show current migration version
	@echo "Showing migration version..."
	@if command -v migrate > /dev/null; then \
		migrate -path migrations -database "$(DATABASE_URL)" version; \
	else \
		echo "migrate not found. Run 'make migrate-install' first."; \
	fi

.DEFAULT_GOAL := help
