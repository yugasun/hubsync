# hubsync

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

## Getting Started

### Option 1: Submit via GitHub Issue

- **Requirement:** Strictly follow the [template](https://github.com/yugasun/hubsync/issues/2) when submitting.
- **Limit:** Up to 11 image addresses per submission.
- **Note:** Docker accounts have daily pull limits. Please use responsibly.

### Option 2: Use GitHub Actions

1. **Bind your DockerHub account:**  
   Go to `Settings` → `Secrets` → `Actions` and add two secrets:

   - `DOCKER_USERNAME` (your Docker username)
   - `DOCKER_PASSWORD` (your Docker password)

2. **Enable Issues:**  
   In `Settings` → `Options` → `Features`, enable the `Issues` feature.

3. **Add Labels:**  
   In `Issues` → `Labels`, add the following labels: `hubsync`, `success`, `failure`.

### Option 3: Run Locally

1. **Clone the repository:**

   ```shell
   git clone https://github.com/yugasun/hubsync
   cd hubsync
   ```

2. **Install dependencies:**

   ```shell
   go mod download
   ```

3. **Build the binary:**

   ```shell
   make build
   ```

4. **Run the sync:**

   ```shell
   ./bin/hubsync --username=xxxxxx --password=xxxxxx --content='{ "hubsync": ["hello-world:latest"] }'
   ```

   **To use a custom image registry:**

   ```shell
   ./bin/hubsync --username=xxxxxx --password=xxxxxx --repository=registry.cn-hangzhou.aliyuncs.com --namespace=xxxxxx --content='{ "hubsync": ["hello-world:latest"] }'
   ```

   **With environment variables:**  
   Create a `.env` file in the project root:

   ```
   DOCKER_USERNAME=your_username
   DOCKER_PASSWORD=your_password
   DOCKER_REPOSITORY=optional_registry_url
   DOCKER_NAMESPACE=your_namespace
   ```

   Then run:

   ```shell
   ./bin/hubsync --content='{ "hubsync": ["hello-world:latest"] }'
   ```

### Option 4: One-line Install (macOS/Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/yugasun/hubsync/main/install.sh | bash
```

> The script will automatically download the latest version of hubsync to /usr/local/bin/hubsync.

## Project Architecture

```
hubsync/
├── cmd/                   # Command-line application entry point
│   └── hubsync/          # Main application command
├── internal/              # Internal packages (not meant to be imported)
│   ├── app/              # Application core logic
│   ├── config/           # Configuration handling
│   └── utils/            # Utilities and helper functions
├── pkg/                   # Public packages that can be imported
│   ├── client/           # Docker client implementation
│   └── sync/             # Image sync functionality
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
make build-all
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
