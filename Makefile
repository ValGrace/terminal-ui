# Command History Tracker Makefile

.PHONY: build test clean install deps fmt vet lint release

# Build variables
BINARY_NAME=tracker
BUILD_DIR=build
DIST_DIR=dist
CMD_DIR=cmd/tracker

# Version information
VERSION?=0.1.0
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'dev')
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-X command-history-tracker/internal/version.Version=$(VERSION) \
        -X command-history-tracker/internal/version.GitCommit=$(GIT_COMMIT) \
        -X command-history-tracker/internal/version.BuildDate=$(BUILD_DATE)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Build the application
build:
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)

# Install the application
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install ./$(CMD_DIR)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run all checks
check: fmt vet test

# Development build (with debug info)
dev-build:
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

# Cross-platform builds
build-all: build-linux build-windows build-darwin

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	GOOS=windows GOARCH=arm64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe ./$(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)

# Release build (all platforms with optimizations)
release: clean
	@echo "Building release v$(VERSION)..."
	@./build.sh --all
	@echo "Release build complete!"

# Create release archives
release-archives: release
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
	for file in tracker-*; do \
		if [ -f "$$file" ]; then \
			tar -czf "$$file.tar.gz" "$$file" && \
			echo "Created $$file.tar.gz"; \
		fi \
	done
	@echo "Release archives created!"

# Help
help:
	@echo "Available targets:"
	@echo "  build            - Build the application"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  clean            - Clean build artifacts"
	@echo "  install          - Install the application"
	@echo "  deps             - Download dependencies"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  check            - Run fmt, vet, and test"
	@echo "  dev-build        - Build with debug info"
	@echo "  build-all        - Build for all platforms"
	@echo "  release          - Build release for all platforms"
	@echo "  release-archives - Create release archives"
	@echo "  help             - Show this help message"