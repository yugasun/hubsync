# HubSync

[![Test](https://github.com/yugasun/hubsync/actions/workflows/test.yml/badge.svg)](https://github.com/yugasun/hubsync/actions/workflows/test.yml)
[![Build](https://github.com/yugasun/hubsync/actions/workflows/release.yml/badge.svg)](https://github.com/yugasun/hubsync/actions/workflows/release.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A tool for accelerating the download of images from foreign registries such as gcr.io, k8s.gcr.io, quay.io, ghcr.io, etc., using docker.io or other mirror services.

> To avoid duplicate requests and make efficient use of resources, please search the issues to see if the image has already been mirrored.

## Features

- Mirror Docker images across different registries
- Concurrent processing for better performance
- Custom image naming support
- Automatic `.env` file loading for easy configuration
- Comprehensive logging and error handling
- Multiple execution modes (CLI, GitHub Actions)
- Multiple installation methods (script, Homebrew, Docker)

## Getting Started

### Option 1: Quick Start Guide

The quickest way to get started is to use our interactive quickstart script:

```sh
curl -fsSL https://raw.githubusercontent.com/yugasun/hubsync/refs/heads/main/quickstart.sh | bash
```

This script will:

- Install HubSync if not already installed
- Guide you through setting up Docker credentials
- Help you run your first sync job

### Option 2: Install and Configure Manually

#### Installation Methods

Choose one of the following installation methods:

**Method A: Direct Install Script (macOS/Linux)**

```sh
curl -fsSL https://raw.githubusercontent.com/yugasun/hubsync/refs/heads/main/install.sh | bash
```

**Method B: Homebrew (macOS)**

```sh
brew tap yugasun/tap
brew install yugasun/tap/hubsync
```

**Method C: Docker**

```sh
docker run -it --rm -v "$(pwd):/data" yugasun/hubsync --help
```

**Method D: Build from Source**

```sh
git clone https://github.com/yugasun/hubsync
cd hubsync
make build
./bin/hubsync --help
```

#### Configuration

Create a `.env` file in your working directory:

```
DOCKER_USERNAME=your_username
DOCKER_PASSWORD=your_password
# Optional settings:
DOCKER_REPOSITORY=your_registry_url
DOCKER_NAMESPACE=your_namespace
```

#### Usage

Basic usage with a single image:

```sh
hubsync --content='{ "hubsync": ["nginx:latest"] }'
```

Advanced usage with multiple options:

```sh
hubsync --username=xxxxxx \
        --password=xxxxxx \
        --repository=registry.cn-hangzhou.aliyuncs.com \
        --namespace=xxxxxx \
        --concurrency=5 \
        --timeout=10m \
        --outputPath=sync.log \
        --logLevel=debug \
        --content='{ "hubsync": ["nginx:latest", "redis:alpine"] }'
```

### Option 3: Submit via GitHub Issue

- **Requirement:** Strictly follow the [template](https://github.com/yugasun/hubsync/issues/2) when submitting.
- **Limit:** Up to 11 image addresses per submission.
- **Note:** Docker accounts have daily pull limits. Please use responsibly.

### Option 4: Use GitHub Actions

1. **Bind your DockerHub account:**  
   Go to `Settings` → `Secrets` → `Actions` and add two secrets:

   - `DOCKER_USERNAME` (your Docker username)
   - `DOCKER_PASSWORD` (your Docker password)

2. **Enable Issues:**  
   In `Settings` → `Options` → `Features`, enable the `Issues` feature.

3. **Add Labels:**  
   In `Issues` → `Labels`, add the following labels: `hubsync`, `success`, `failure`.

## Docker Support

HubSync is available as a Docker image for containerized environments:

```sh
# Pull the image
docker pull yugasun/hubsync:latest

# Run using environment variables
docker run -it --rm \
  -v "$(pwd):/data" \
  -e DOCKER_USERNAME=your_username \
  -e DOCKER_PASSWORD=your_password \
  yugasun/hubsync --content='{ "hubsync": ["nginx:latest"] }'

# Or run using a local .env file
docker run -it --rm \
  -v "$(pwd):/data" \
  -v "$(pwd)/.env:/data/.env" \
  yugasun/hubsync --content='{ "hubsync": ["nginx:latest"] }'
```

## Project Architecture

```
hubsync/
├── cmd/                   # Command-line application entry point
│   └── hubsync/          # Main application command
├── internal/              # Internal packages (not meant to be imported)
│   ├── app/              # Application core logic
│   ├── config/           # Configuration handling
│   ├── di/               # Dependency injection container
│   └── utils/            # Utilities and helper functions
├── pkg/                   # Public packages that can be imported
│   ├── docker/           # Docker client implementation
│   ├── errors/           # Error handling and custom error types
│   ├── observability/    # Metrics and telemetry
│   ├── registry/         # Registry client interfaces and implementations
│   └── sync/             # Image sync functionality
│       └── strategies/   # Synchronization strategies (standard/parallel)
└── test/                  # Test files and mocks
    ├── integration/      # Integration tests
    ├── mocks/            # Mock implementations
    └── unit/             # Unit tests
```

## Development Guide

### Prerequisites

- Go 1.18 or higher
- Docker (for running integration tests)
- Make

### Setup Development Environment

1. **Clone the repository:**

   ```shell
   git clone https://github.com/yugasun/hubsync.git
   cd hubsync
   ```

2. **Install dependencies:**

   ```shell
   go mod download
   ```

3. **Create a local .env file:**

   ```shell
   cat > .env << EOF
   DOCKER_USERNAME=your_username
   DOCKER_PASSWORD=your_password
   # Optional settings:
   # DOCKER_REPOSITORY=your_registry_url
   # DOCKER_NAMESPACE=your_namespace
   # LOG_LEVEL=debug
   EOF
   ```

### Running Tests

```shell
# Run unit tests
make unit-test

# Run integration tests (requires Docker credentials)
make integration-test

# Run all tests
make test

# Check test coverage
make cover
```

### Building the Application

```shell
# Build for current platform
make build

# Build for all supported platforms
make cross
```

### Code Structure

- **Interface-driven design:** Components use interfaces for better testability
- **Clean Architecture:** Core business logic is separated from external dependencies
- **Configuration:** Supports CLI flags, environment variables, and .env files

### Contributing

1. Fork the repository
2. Create your feature branch: `git checkout -b feature/my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Push to the branch: `git push origin feature/my-new-feature`
5. Submit a pull request

## License

MIT @yugasun
