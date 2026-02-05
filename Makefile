.PHONY: all build install install-local clean test lint fmt check help release dist

# Binary name
BINARY := pocket

# Build directory
BUILD_DIR := ./build

# Install directory (user-local, no sudo needed)
INSTALL_DIR := $(HOME)/.local/bin

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Version (from git tag or default)
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")

# Build flags
LDFLAGS := -s -w -X main.Version=$(VERSION)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Default target
all: check build

## build: Build the binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/pocket

## install: Install locally for development
install: build
	@echo "Installing $(BINARY) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@chmod +x $(INSTALL_DIR)/$(BINARY)
	@echo "✅ Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo ""
	@if [[ ":$$PATH:" != *":$(INSTALL_DIR):"* ]]; then \
		echo "⚠️  Add to your PATH:"; \
		echo "  export PATH=\"\$$PATH:$(INSTALL_DIR)\""; \
		echo ""; \
	fi

## install-local: Full local install with PATH setup (dev)
install-local:
	@./install.sh

## release: Build release binaries for all platforms
release:
	@chmod +x scripts/release.sh
	@./scripts/release.sh $(VERSION)

## dist: Alias for release
dist: release

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf ./dist
	@rm -f coverage.out

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-short: Run tests without race detector
test-short:
	@echo "Running tests (short)..."
	$(GOTEST) -v ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w -local github.com/unstablemind/pocket .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## check: Run all checks (fmt, vet, lint)
check: fmt vet lint

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## run: Build and run
run: build
	@$(BUILD_DIR)/$(BINARY)

## tag: Create a new version tag
tag:
	@echo "Current version: $(VERSION)"
	@read -p "New version (e.g., v0.2.0): " new_version; \
	git tag -a $$new_version -m "Release $$new_version"; \
	echo "Created tag $$new_version"; \
	echo "Push with: git push origin $$new_version"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  build         Build the binary"
	@echo "  install       Install locally (to ~/.local/bin)"
	@echo "  run           Build and run"
	@echo "  clean         Remove build artifacts"
	@echo ""
	@echo "Quality:"
	@echo "  check         Run all checks (fmt, vet, lint)"
	@echo "  test          Run tests"
	@echo "  lint          Run golangci-lint"
	@echo "  fmt           Format code"
	@echo ""
	@echo "Release:"
	@echo "  release       Build binaries for all platforms"
	@echo "  tag           Create a new version tag"
	@echo ""
	@echo "Install from GitHub (for users):"
	@echo "  curl -fsSL https://raw.githubusercontent.com/KenKaiii/pocket-agent-cli/main/scripts/install.sh | bash"
