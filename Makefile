# Makefile for safeguard

.PHONY: all build test test-verbose test-coverage clean install help bazel-build bazel-test bazel-clean bazel-run

# Default target
all: test build

# Build the application
build:
	go build -o safeguard.exe ./cmd/cli

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=windows GOARCH=amd64 go build -o dist/safeguard-windows-amd64.exe ./cmd/cli
	GOOS=darwin GOARCH=amd64 go build -o dist/safeguard-darwin-amd64 ./cmd/cli
	GOOS=darwin GOARCH=arm64 go build -o dist/safeguard-darwin-arm64 ./cmd/cli
	GOOS=linux GOARCH=amd64 go build -o dist/safeguard-linux-amd64 ./cmd/cli
	GOOS=linux GOARCH=arm64 go build -o dist/safeguard-linux-arm64 ./cmd/cli
	@echo "Build complete!"

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	go test ./... -race

# Run benchmarks
bench:
	go test ./... -bench=. -benchmem

# Clean build artifacts
clean:
	rm -f safeguard.exe safeguard-svc.exe
	rm -f coverage.out coverage.html
	rm -rf dist/

# Build the builder UI (requires Node.js)
build-ui:
	cd cmd/builder/ui && npm ci && npm run build

# Build the builder server (builds UI first)
build-builder: build-ui
	go build -o builder.exe ./cmd/builder/

# Build with Bazel
bazel-build:
	bazel build //:safeguard

# Run with Bazel
bazel-run:
	bazel run //:safeguard

# Test with Bazel
bazel-test:
	bazel test //...

# Clean Bazel artifacts
bazel-clean:
	bazel clean

# Deep clean Bazel (removes all caches)
bazel-expunge:
	bazel clean --expunge

# Install dependencies
install:
	go mod download
	go mod tidy

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
"
	@echo "Bazel targets:"
	@echo "  make bazel-build    - Build with Bazel"
	@echo "  make bazel-run      - Run with Bazel"
	@echo "  make bazel-test     - Test with Bazel"
	@echo "  make bazel-clean    - Clean Bazel artifacts"
	@echo "  make bazel-expunge  - Deep clean Bazel (removes all caches)"
	@echo ""
	@echo "
# Run the application (dev mode)
run:
	go run ./cmd/cli -mount V: -auth-method token -vault-token dev-token -debug

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the application"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make test           - Run tests"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make bench          - Run benchmarks"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install dependencies"
	@echo "  make lint           - Run linter"
	@echo "  make fmt            - Format code"
	@echo "  make run            - Run the application in dev mode"
	@echo "  make help           - Show this help message"
