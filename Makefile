# Makefile for floatx80 Go library

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Package info
PACKAGE=./...
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

.PHONY: all build clean test bench coverage deps tidy fmt vet lint help

.PHONY: all build clean test bench coverage deps tidy fmt vet lint help

# Default target
all: fmt vet test

# Build/verify the project compiles
build:
	$(GOBUILD) $(PACKAGE)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) $(PACKAGE)
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)

# Run benchmarks
bench:
	$(GOTEST) -bench=. -benchmem ./...

# Run benchmarks with memory profiling
bench-profile:
	$(GOTEST) -bench=. -benchmem -memprofile=mem.prof ./...

# Download dependencies
deps:
	$(GOMOD) download

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Format code
fmt:
	$(GOFMT) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Run golint (if available)
lint:
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint not installed. Install with: go install golang.org/x/lint/golint@latest"; \
	fi

# Run staticcheck (if available)
staticcheck:
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

# Run all checks (fmt, vet, lint, staticcheck)
check: fmt vet lint staticcheck

# Generate documentation
doc:
	$(GOCMD) doc -all > doc.txt

# Run tests and generate coverage report
test-coverage: test coverage

# Development workflow: format, vet, test, build
dev: fmt vet test build

# CI workflow: tidy, check, test, build
ci: tidy check test build

# Install development tools
install-tools:
	$(GOCMD) install golang.org/x/lint/golint@latest
	$(GOCMD) install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOCMD) install golang.org/x/tools/cmd/goimports@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Run fmt, vet, and test"
	@echo "  build        - Verify the project compiles"
	@echo "  clean        - Clean build artifacts and coverage files"
	@echo "  test         - Run tests"
	@echo "  bench        - Run benchmarks"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy dependencies"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golint (if installed)"
	@echo "  staticcheck  - Run staticcheck (if installed)"
	@echo "  check        - Run all code quality checks"
	@echo "  doc          - Generate documentation"
	@echo "  dev          - Development workflow (fmt, vet, test)"
	@echo "  ci           - CI workflow (tidy, check, test)"
	@echo "  install-tools- Install development tools"
	@echo "  help         - Show this help message"