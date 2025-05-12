# HubSync Testing

This directory contains the test code for the HubSync project, organized into different categories to ensure comprehensive test coverage.

## Test Structure

- **Unit Tests** (`/test/unit/`): Tests individual components in isolation
  - `config_test.go`: Tests for configuration handling
  - `content_parser_test.go`: Tests for JSON content parsing
  - `models_test.go`: Tests for data structures
  - `name_generator_test.go`: Tests for image name generation functionality
  - `syncer_test.go`: Tests for the core synchronization functionality

- **Mocks** (`/test/mocks/`): Mock implementations for testing
  - `docker_client.go`: A mock implementation of the Docker client

- **Integration Tests** (`/test/integration/`): End-to-end tests that use real Docker operations
  - `integration_test.go`: Tests that perform actual Docker registry operations

## Running Tests

### Unit Tests

Run unit tests with:

```bash
make unit-test
```

These tests use mock implementations and don't require Docker credentials or internet access.

### Integration Tests 

Run integration tests with:

```bash
make integration-test
```

**Note**: Integration tests require:
- Docker credentials defined in a `.env` file
- Access to Docker registries
- Internet connectivity

### All Tests

Run all tests (unit + integration) with:

```bash
make test
```

## Test Environment Setup

1. Create a `.env` file in the project root with your Docker credentials:

```
DOCKER_USERNAME=your_username
DOCKER_PASSWORD=your_password
DOCKER_REPOSITORY=optional_private_registry_url
```

2. Ensure Docker daemon is running locally

## Writing New Tests

When adding new functionality:

1. Create unit tests in the appropriate file in `/test/unit/`
2. Update or extend mock implementations in `/test/mocks/` if needed
3. Add integration tests in `/test/integration/` for end-to-end validation
4. Make sure to run both unit and integration tests before submitting changes

## Test Coverage

Check test coverage with:

```bash
make cover
```

This will generate a coverage report and open it in your browser.