.PHONY: all build test clean lint fmt vet install examples coverage bench

# Default target
all: clean fmt vet lint test build

# Build the library
build:
	go build -v ./...

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Clean build artifacts
clean:
	go clean
	rm -f coverage.out coverage.html
	rm -f examples/quick_start/quick_start
	rm -f examples/interactive_client/interactive_client
	rm -f examples/streaming_mode/streaming_mode

# Format code
fmt:
	go fmt ./...
	gofmt -s -w .

# Run go vet
vet:
	go vet ./...

# Run golint (requires: go install golang.org/x/lint/golint@latest)
lint:
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; \
	fi

# Install the library locally
install:
	go install ./...

# Build examples
examples: build
	go build -o examples/quick_start/quick_start ./examples/quick_start
	go build -o examples/interactive_client/interactive_client ./examples/interactive_client
	go build -o examples/streaming_mode/streaming_mode ./examples/streaming_mode

# Run examples
run-examples: examples
	@echo "Running quick start example..."
	./examples/quick_start/quick_start
	@echo "\nStreaming mode example..."
	./examples/streaming_mode/streaming_mode

# Development setup
setup:
	go mod download
	go install golang.org/x/lint/golint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development environment setup complete"

# Check if code is ready for commit
check: fmt vet lint test
	@echo "All checks passed!"

# Update dependencies
deps:
	go mod tidy
	go mod verify

# Generate documentation
docs:
	@echo "Opening documentation in browser..."
	@go doc -http=:6060 &
	@sleep 2
	@open http://localhost:6060/pkg/github.com/davlia/claude-code-sdk-go/

# Help target
help:
	@echo "Available targets:"
	@echo "  all        - Format, vet, lint, test, and build"
	@echo "  build      - Build the library"
	@echo "  test       - Run tests with race detection"
	@echo "  coverage   - Generate test coverage report"
	@echo "  bench      - Run benchmarks"
	@echo "  clean      - Remove build artifacts"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run golint"
	@echo "  install    - Install the library"
	@echo "  examples   - Build example programs"
	@echo "  setup      - Setup development environment"
	@echo "  check      - Run all checks (fmt, vet, lint, test)"
	@echo "  deps       - Update dependencies"
	@echo "  docs       - Open documentation server"
	@echo "  help       - Show this help message"