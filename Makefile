.PHONY: all build install clean test lint fmt check help

# Binary name
BINARY := pocket

# Build directory
BUILD_DIR := ./build

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -s -w
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Default target
all: check build

## build: Build the binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/pocket

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY)..."
	$(GOBUILD) $(BUILD_FLAGS) -o $(GOPATH)/bin/$(BINARY) ./cmd/pocket

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
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

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'
