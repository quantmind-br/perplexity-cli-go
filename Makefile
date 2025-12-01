# Perplexity CLI - Makefile
# Build and install commands for the Go CLI

# Variables
BINARY_NAME := perplexity
VERSION := 1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Directories
BUILD_DIR := ./build
GOPATH_BIN := $(shell go env GOPATH)/bin

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOMOD := $(GOCMD) mod

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build install uninstall clean test test-coverage deps tidy cross-compile help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/perplexity
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build optimized release binary
build-release:
	@echo "Building optimized release binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -ldflags "-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/perplexity
	@echo "Release binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH_BIN)..."
	@mkdir -p $(GOPATH_BIN)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH_BIN)/$(BINARY_NAME)
	@chmod +x $(GOPATH_BIN)/$(BINARY_NAME)
	@echo "Installed: $(GOPATH_BIN)/$(BINARY_NAME)"
	@echo "Run '$(BINARY_NAME) --help' to get started"

# Install to user directory (no sudo required)
install-user: build
	@echo "Installing $(BINARY_NAME) to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	@chmod +x ~/.local/bin/$(BINARY_NAME)
	@echo "Installed: ~/.local/bin/$(BINARY_NAME)"
	@echo "Make sure ~/.local/bin is in your PATH"

# Uninstall from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(GOPATH_BIN)..."
	@rm -f $(GOPATH_BIN)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Uninstall from user directory
uninstall-user:
	@echo "Uninstalling $(BINARY_NAME) from ~/.local/bin..."
	@rm -f ~/.local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out
	@rm -f coverage.out

# Run tests with HTML coverage report
test-coverage-html:
	@echo "Generating HTML coverage report..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied"

# Cross-compile for all platforms
cross-compile:
	@echo "Cross-compiling for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$(if [ "$${platform%/*}" = "windows" ]; then echo ".exe"; fi) ./cmd/perplexity; \
		echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done
	@echo "Cross-compilation complete"

# Build for Linux amd64
build-linux:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/perplexity
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

# Build for macOS (Apple Silicon)
build-darwin:
	@echo "Building for macOS arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/perplexity
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for Windows
build-windows:
	@echo "Building for Windows amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/perplexity
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Run the binary
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Help
help:
	@echo "Perplexity CLI - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build           Build the binary (default)"
	@echo "  build-release   Build optimized release binary"
	@echo "  cross-compile   Build for all platforms"
	@echo "  build-linux     Build for Linux amd64"
	@echo "  build-darwin    Build for macOS arm64"
	@echo "  build-windows   Build for Windows amd64"
	@echo ""
	@echo "Install targets:"
	@echo "  install         Install to GOPATH/bin"
	@echo "  install-user    Install to ~/.local/bin"
	@echo "  uninstall       Remove from GOPATH/bin"
	@echo "  uninstall-user  Remove from ~/.local/bin"
	@echo ""
	@echo "Test targets:"
	@echo "  test            Run all tests"
	@echo "  test-coverage   Run tests with coverage summary"
	@echo "  test-coverage-html  Generate HTML coverage report"
	@echo ""
	@echo "Other targets:"
	@echo "  clean           Remove build artifacts"
	@echo "  deps            Download dependencies"
	@echo "  tidy            Tidy go.mod"
	@echo "  run ARGS='...'  Build and run with arguments"
	@echo "  version         Show version info"
	@echo "  help            Show this help"
