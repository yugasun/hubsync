.PHONY: build run test integration-test clean lint vet cover help all lint-install lint-fix lint-ci

BINARY_NAME=hubsync
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/yugasun/hubsync/cmd/hubsync.Version=$(VERSION)"
BUILD_DIR=bin
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
MAIN_PACKAGE=./cmd/hubsync

all: clean lint test build

help:
	@echo "HubSync - Docker Hub Image Synchronization Tool"
	@echo ""
	@echo "Usage:"
	@echo "  make build         Build for current platform"
	@echo "  make run           Run the application with options:"
	@echo "                     image=\"nginx:latest\"    : Specify a single image"
	@echo "                     content=\"{...}\"         : Specify full JSON content"
	@echo "                     username=\"user\"         : Docker username"
	@echo "                     password=\"pass\"         : Docker password/token"
	@echo "                     repository=\"repo.com\"   : Docker repository"
	@echo "                     namespace=\"ns\"          : Custom namespace"
	@echo "                     concurrency=5            : Number of concurrent operations"
	@echo "                     timeout=\"10m\"           : Operation timeout"
	@echo "                     output=\"file.log\"       : Output file path"
	@echo "                     loglevel=\"debug\"        : Log level (debug, info, warn, error)"
	@echo "  make test          Run all tests"
	@echo "  make unit-test     Run only unit tests"
	@echo "  make integration-test Run integration tests"
	@echo "  make cross         Build for all supported platforms"
	@echo "  make clean         Remove build artifacts"
	@echo "  make lint          Run linters"
	@echo "  make lint-fix      Run linters and fix issues automatically"
	@echo "  make lint-ci       Run linters in CI mode (strict)"
	@echo "  make lint-install  Install linting tools"
	@echo "  make vet           Run go vet"
	@echo "  make cover         Run tests with coverage"
	@echo "  make all           Clean, lint, test, and build"

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

# Enhanced run command with more flexible options
run:
	@if [ -f .env ]; then \
		echo "Loading environment from .env file..."; \
		. ./.env; \
	fi; \
	if [ -n "$(image)" ] && [ -z "$(content)" ]; then \
		echo "Running with image: $(image)"; \
		CONTENT='{ "hubsync": ["$(image)"] }'; \
	else \
		CONTENT='$(content)'; \
	fi; \
	USERNAME=$${username:-$$DOCKER_USERNAME}; \
	PASSWORD=$${password:-$$DOCKER_PASSWORD}; \
	REPOSITORY=$${repository:-$$DOCKER_REPOSITORY}; \
	NAMESPACE=$${namespace:-"yugasun"}; \
	CONCURRENCY=$${concurrency:-3}; \
	TIMEOUT=$${timeout:-"10m"}; \
	OUTPUT=$${output:-"output.log"}; \
	LOGLEVEL=$${loglevel:-"info"}; \
	echo "Starting hubsync with configuration:"; \
	echo "- Repository: $${REPOSITORY:-Docker Hub}"; \
	echo "- Namespace:  $${NAMESPACE}"; \
	echo "- Concurrency: $${CONCURRENCY}"; \
	echo "- Timeout: $${TIMEOUT}"; \
	echo "- Output: $${OUTPUT}"; \
	echo "- Log Level: $${LOGLEVEL}"; \
	go run $(LDFLAGS) $(MAIN_PACKAGE)/main.go \
		--username="$${USERNAME}" \
		--password="$${PASSWORD}" \
		--repository="$${REPOSITORY}" \
		--namespace="$${NAMESPACE}" \
		--concurrency="$${CONCURRENCY}" \
		--timeout="$${TIMEOUT}" \
		--outputPath="$${OUTPUT}" \
		--logLevel="$${LOGLEVEL}" \
		--content="$${CONTENT}"

test:
	@echo "Running all tests..."
	go test -v ./...

unit-test:
	@echo "Running unit tests..."
	go test -v ./pkg/... ./cmd/... ./internal/...

integration-test:
	@echo "Running integration tests..."
	@if [ -f .env ]; then \
		. ./.env; \
	fi; \
	go test -v -tags=integration ./test/integration/...

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

lint-install:
	@echo "Installing linting tools..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint already installed"; \
	elif command -v brew >/dev/null 2>&1; then \
		echo "Installing golangci-lint via Homebrew..."; \
		brew install golangci-lint; \
	elif [ "$(shell uname)" = "Darwin" ]; then \
		echo "macOS detected. You can install golangci-lint via:"; \
		echo "  brew install golangci-lint"; \
		echo "or manually download from https://github.com/golangci/golangci-lint/releases"; \
		exit 1; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Linting tools installed successfully!"

lint: lint-install
	@echo "Running linters..."
	golangci-lint run --timeout=5m ./...

lint-fix: lint-install
	@echo "Running linters with auto-fix..."
	golangci-lint run --fix ./...

lint-ci: lint-install
	@echo "Running linters in CI mode (strict)..."
	golangci-lint run --timeout=5m --out-format=github-actions ./... || exit 1

vet:
	@echo "Running go vet..."
	go vet ./...

cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

cross:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	$(foreach platform,$(PLATFORMS),\
		$(eval GOOS=$(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform)))) \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(if $(findstring windows,$(GOOS)),.exe,) $(MAIN_PACKAGE) ; \
	)
	@echo "Done!"

# Default target
.DEFAULT_GOAL := help