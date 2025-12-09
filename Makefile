.PHONY: build install clean test lint run-daemon run-cli

# Build variables
BINARY_DAEMON=octa-agentd
BINARY_CLI=octa-agent
BUILD_DIR=./bin
CMD_DAEMON=./cmd/octa-agentd
CMD_CLI=./cmd/octa-agent

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
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run

run-daemon: build-daemon
	@echo "Starting octa-agentd..."
	$(BUILD_DIR)/$(BINARY_DAEMON)

run-cli: build-cli
	@echo "Running octa-agent..."
	$(BUILD_DIR)/$(BINARY_CLI)

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
	@echo "  build         - Build all binaries"
	@echo "  build-daemon  - Build octa-agentd"
	@echo "  build-cli     - Build octa-agent"
	@echo "  install       - Install binaries to GOPATH"
	@echo "  clean         - Remove build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  run-daemon    - Build and run daemon"
	@echo "  run-cli       - Build and run CLI"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  vet           - Vet code"
