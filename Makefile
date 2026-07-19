.PHONY: build install clean test lint run-daemon run-cli test-coverage coverage-gate live-test

# Build variables
BINARY_DAEMON=octa-agentd
BINARY_CLI=octa-agent
BUILD_DIR=./bin
CMD_DAEMON=./cmd/octa-agentd
CMD_CLI=./cmd/octa-agent
COVERAGE_MIN ?= 20

# Build flags
LDFLAGS=-ldflags "-s -w"

all: build

build: build-daemon build-cli

build-daemon:
	@echo "Building octa-agentd..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_DAEMON) $(CMD_DAEMON)

build-cli:
	@echo "Building octa-agent..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_CLI) $(CMD_CLI)

install: build
	@echo "Installing binaries..."
	go install $(CMD_DAEMON)
	go install $(CMD_CLI)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Fail if total statement coverage is below COVERAGE_MIN (default 20%).
coverage-gate: test-coverage
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {print $$3}' | tr -d '%'); \
	echo "Total coverage: $${total}% (min $(COVERAGE_MIN)%)"; \
	awk -v t="$$total" -v m="$(COVERAGE_MIN)" 'BEGIN { exit (t+0 < m+0) }' || \
		(echo "coverage $${total}% is below gate $(COVERAGE_MIN)%" && exit 1)

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed; running go vet + gofmt check"; \
		go vet ./...; \
		test -z "$$(gofmt -l . | grep -v vendor || true)" || (gofmt -l . && exit 1); \
	fi

run-daemon: build-daemon
	@echo "Starting octa-agentd..."
	$(BUILD_DIR)/$(BINARY_DAEMON)

run-cli: build-cli
	@echo "Running octa-agent..."
	$(BUILD_DIR)/$(BINARY_CLI)

# End-to-end smoke against a live Ollama instance (not for CI by default).
live-test: build
	@echo "Running live end-to-end smoke test..."
	@./scripts/live-test.sh

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Vetting code..."
	go vet ./...

help:
	@echo "Available targets:"
	@echo "  build          - Build all binaries"
	@echo "  build-daemon   - Build octa-agentd"
	@echo "  build-cli      - Build octa-agent"
	@echo "  install        - Install binaries to GOPATH"
	@echo "  clean          - Remove build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  coverage-gate  - Fail if coverage < COVERAGE_MIN (default 20)"
	@echo "  lint           - Run linter (golangci-lint or vet/gofmt)"
	@echo "  run-daemon     - Build and run daemon"
	@echo "  run-cli        - Build and run CLI"
	@echo "  live-test      - E2E smoke (daemon+CLI+Ollama; requires ollama/sqlite3)"
	@echo "  deps           - Download dependencies"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
