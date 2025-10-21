# EdgeLink Desktop Client Makefile
# Cross-platform build targets for EdgeLink desktop client

# Variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
COMMIT_SHA ?= $(shell git rev-parse --short=8 HEAD 2>/dev/null || echo 'unknown')
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_DIR := $(CURDIR)/../../build/clients
BUILD_SCRIPT := $(CURDIR)/../../scripts/build-desktop-clients.sh

# Go build variables
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
GO_LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.CommitSHA=$(COMMIT_SHA) \
	-X main.BuildDate=$(BUILD_DATE)

# Binary name
BINARY_NAME := edgelink-client

# Source files
SRC_DIR := $(CURDIR)
CMD_DIR := $(SRC_DIR)/cmd/edgelink-client

# Colors for output
COLOR_RESET := \033[0m
COLOR_GREEN := \033[0;32m
COLOR_BLUE := \033[0;34m
COLOR_YELLOW := \033[1;33m

# Phony targets
.PHONY: all build build-all build-linux build-windows build-macos clean test lint help install uninstall

# Default target
all: build

## build: Build for current platform
build:
	@echo "$(COLOR_GREEN)Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build \
		-trimpath \
		-ldflags "$(GO_LDFLAGS)" \
		-o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)-$(VERSION) \
		$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)-$(VERSION)$(COLOR_RESET)"

## build-all: Build for all supported platforms
build-all:
	@echo "$(COLOR_BLUE)Building for all platforms...$(COLOR_RESET)"
	@$(BUILD_SCRIPT) --version $(VERSION)

## build-linux: Build for Linux (amd64, arm64)
build-linux:
	@echo "$(COLOR_BLUE)Building for Linux platforms...$(COLOR_RESET)"
	@$(BUILD_SCRIPT) --version $(VERSION) --platform linux/amd64,linux/arm64

## build-windows: Build for Windows (amd64, arm64)
build-windows:
	@echo "$(COLOR_BLUE)Building for Windows platforms...$(COLOR_RESET)"
	@$(BUILD_SCRIPT) --version $(VERSION) --platform windows/amd64,windows/arm64

## build-macos: Build for macOS (amd64, arm64)
build-macos:
	@echo "$(COLOR_BLUE)Building for macOS platforms...$(COLOR_RESET)"
	@$(BUILD_SCRIPT) --version $(VERSION) --platform darwin/amd64,darwin/arm64

## build-parallel: Build all platforms in parallel
build-parallel:
	@echo "$(COLOR_BLUE)Building all platforms in parallel...$(COLOR_RESET)"
	@$(BUILD_SCRIPT) --version $(VERSION) --parallel

## clean: Clean build artifacts
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@echo "$(COLOR_GREEN)Cleaned successfully$(COLOR_RESET)"

## test: Run tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	@go test -v -race -cover ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)Coverage report generated: coverage.html$(COLOR_RESET)"

## lint: Run linters
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint not found, skipping...$(COLOR_RESET)"; \
		go vet ./...; \
	fi

## fmt: Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./...
	@gofmt -s -w .

## install: Install binary to local system
install: build
	@echo "$(COLOR_BLUE)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)-$(VERSION) /usr/local/bin/$(BINARY_NAME)
	@echo "$(COLOR_GREEN)Installed to /usr/local/bin/$(BINARY_NAME)$(COLOR_RESET)"

## uninstall: Uninstall binary from local system
uninstall:
	@echo "$(COLOR_YELLOW)Uninstalling $(BINARY_NAME)...$(COLOR_RESET)"
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(COLOR_GREEN)Uninstalled successfully$(COLOR_RESET)"

## deps: Download and tidy dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	@go mod download
	@go mod tidy
	@echo "$(COLOR_GREEN)Dependencies updated$(COLOR_RESET)"

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_SHA)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(shell go version)"

## help: Show this help message
help:
	@echo "EdgeLink Desktop Client Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION     - Version tag (default: git describe)"
	@echo "  BUILD_DIR   - Output directory (default: ../../build/clients)"
	@echo ""
	@echo "Examples:"
	@echo "  make build                    # Build for current platform"
	@echo "  make build-all                # Build for all platforms"
	@echo "  make build-linux              # Build for Linux only"
	@echo "  make VERSION=v1.2.3 build-all # Build with custom version"
	@echo "  make test                     # Run tests"
	@echo "  make clean                    # Clean build artifacts"
