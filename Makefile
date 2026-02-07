.PHONY: all build test lint clean install release help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -ldflags "-X github.com/richgo/flo/cmd/flo/cmd.version=$(VERSION) -X github.com/richgo/flo/cmd/flo/cmd.commit=$(COMMIT) -X github.com/richgo/flo/cmd/flo/cmd.date=$(DATE)"

# Default target
all: lint test build

# Build the project
build:
	@echo "Building flo..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/flo ./cmd/flo
	@echo "Build complete: bin/flo"

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@echo "Clean complete"

# Install the binary
install:
	@echo "Installing flo..."
	@go install $(LDFLAGS) ./cmd/flo
	@echo "Install complete"

# Create a release
release:
	@echo "Creating release..."
	@goreleaser release --clean
	@echo "Release complete"

# Show help
help:
	@echo "Flo Makefile targets:"
	@echo "  all      - Run lint, test, and build (default)"
	@echo "  build    - Build the flo binary to bin/flo"
	@echo "  test     - Run all tests"
	@echo "  coverage - Run tests with coverage report"
	@echo "  lint     - Run golangci-lint"
	@echo "  clean    - Remove build artifacts"
	@echo "  install  - Install flo to GOPATH/bin"
	@echo "  release  - Create a release with goreleaser"
	@echo "  help     - Show this help message"
