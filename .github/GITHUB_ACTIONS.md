# GitHub Actions Workflows

This document describes the GitHub Actions workflows used in the HubSync project and provides guidelines for contributors looking to modify or extend these workflows.

## Available Workflows

### 1. Test Workflow (`test.yml`)

Triggered on push to `main` branch and pull requests to `main`.

**Purpose:** Ensures code quality and test coverage.

**Jobs:**
- **Lint**: Runs `golangci-lint` to check code quality
- **Unit Tests**: Runs all unit tests with `make unit-test`
- **Code Coverage**: Uploads test coverage report to Codecov

**Usage for Contributors:**
- All PRs must pass this workflow before merging
- Keep code coverage above 75%
- Fix all linting issues before submitting PR

### 2. Release Workflow (`release.yml`)

Triggered when a tag with the prefix `v` is pushed (e.g., `v1.0.0`).

**Purpose:** Creates GitHub releases and builds binaries for multiple platforms.

**Jobs:**
- **Test**: Validates that all tests pass
- **Build**: Compiles binaries for multiple OS/architecture combinations
- **Release**: Creates a GitHub release with generated binaries

**Usage for Contributors:**
- Tag format must be `vX.Y.Z` following semantic versioning
- Release notes are auto-generated from commit messages
- Update version references in code before tagging

### 3. Issue Sync Workflow (`issue-sync.yml`)

Triggered when issues are created or edited with the `hubsync` label or `[hubsync]` in the title.

**Purpose:** Processes image sync requests from issues.

**Jobs:**
- **Sync**: Extracts image list from issue body and runs HubSync
- Adds `success` or `failure` labels based on the outcome
- Posts a comment with the sync results

**Usage for Contributors:**
- Do not modify this workflow without testing with actual issues
- Ensure backward compatibility with existing issue formats
- Add proper error handling for edge cases

## Environment Variables and Secrets

The following secrets should be configured in your repository settings:

- `DOCKER_USERNAME`: DockerHub username for image operations
- `DOCKER_PASSWORD`: DockerHub password or token
- `DOCKER_REPOSITORY`: (Optional) Custom container registry
- `DOCKER_NAMESPACE`: (Optional) Custom namespace for images
- `GH_TOKEN`: GitHub token for creating releases
- `CODECOV_TOKEN`: Token for uploading coverage reports to Codecov

## Development Guidelines for Workflows

### Testing Workflow Changes

Before submitting workflow changes:

1. Fork the repository and create a feature branch
2. Make your changes to the workflow file
3. Push to your fork and verify that the workflow runs correctly
4. If possible, test different scenarios (success, failure, etc.)

### Workflow Best Practices

- Keep workflows focused on a single responsibility
- Use consistent naming for steps and jobs
- Set appropriate timeouts to avoid hung jobs
- Cache dependencies to speed up builds
- Use continue-on-error for non-critical steps
- Document secrets and environment variables

### Security Considerations

- Avoid logging sensitive information
- Use secrets for all credentials
- Limit permissions of GitHub tokens
- Be cautious with user-provided input (e.g., from issues)

## Troubleshooting

If a workflow fails:

1. Check the detailed logs in the GitHub Actions tab
2. Verify that all secrets are correctly configured
3. Try running the same commands locally
4. For issue sync failures, verify the JSON format in the issue

## Extending the Workflows

When adding new features:

1. Consider adding appropriate tests in the `test.yml` workflow
2. Update the release workflow if new artifacts need to be published
3. Document any new environment variables or secrets
4. Consider backward compatibility with existing CI/CD pipelines

For major changes to workflows, please open an issue first to discuss your approach.