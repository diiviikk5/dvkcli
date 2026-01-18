# dvkcli - Local-first AI terminal assistant
# Makefile for building and development

BINARY_NAME=dvkcli
VERSION=0.1.0
BUILD_DIR=build
MAIN_PATH=./cmd/dvkcli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build clean test deps run install

all: deps build

# Download dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"

# Build for release (optimized)
release:
	@echo "Building release version..."
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)

# Run the application
run:
	$(GOCMD) run $(MAIN_PATH)

# Install to GOPATH/bin
install:
	$(GOCMD) install $(LDFLAGS) $(MAIN_PATH)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Format code
fmt:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Development mode with live reload (requires air)
dev:
	air

# Show help
help:
	@echo "Available commands:"
	@echo "  make deps      - Download dependencies"
	@echo "  make build     - Build the binary"
	@echo "  make run       - Run the application"
	@echo "  make install   - Install to GOPATH/bin"
	@echo "  make test      - Run tests"
	@echo "  make clean     - Clean build artifacts"
	@echo "  make fmt       - Format code"
	@echo "  make help      - Show this help"
