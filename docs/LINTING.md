# Linting in HubSync

This document provides information about the linting setup and practices for the HubSync project.

## Overview

We use [golangci-lint](https://github.com/golangci/golangci-lint) as our primary linting tool. It's a fast, customizable Go linter that bundles multiple linters together, providing a comprehensive code quality check.

## Enabled Linters

Our configuration includes the following linters:

- `errcheck` - Check for unchecked errors
- `gosimple` - Suggests code simplifications
- `govet` - Reports suspicious code constructs
- `ineffassign` - Detects unused assignments
- `staticcheck` - Static analysis checks
- `typecheck` - Type checking
- `unused` - Checks for unused constants, variables, functions, and types
- `gosec` - Security checks
- `gofmt` - Checks whether code was gofmt-ed
- `goimports` - Checks import statements formatting
- `revive` - Fast, configurable, extensible linter for Go
- `misspell` - Finds commonly misspelled English words
- `bodyclose` - Checks whether HTTP response body is closed
- `dupl` - Finds code clones
- `unconvert` - Removes unnecessary type conversions
- `unparam` - Reports unused function parameters
- `nolintlint` - Reports bad usage of nolint directives
- `prealloc` - Finds slice declarations that could be pre-allocated

## Usage

### Regular Linting

To run the linters on your code:

```bash
make lint
```

This will install golangci-lint if needed and run it with our configuration.

### Auto-fix Linting Issues

Some linting issues can be automatically fixed:

```bash
make lint-fix
```

### Install Linting Tools Only

To just install the linting tools without running them:

```bash
make lint-install
```

### CI Mode

A stricter version of linting used for CI pipelines:

```bash
make lint-ci
```

## Git Pre-commit Hook

A pre-commit hook is included to run linting checks before each commit. This helps catch issues early in development.

To bypass the pre-commit hook in rare cases:

```bash
git commit --no-verify -m "Your commit message"
```

## Suppressing Linting Issues

If you need to suppress a linting issue for a valid reason, use inline directives:

```go
// nolint:lintername // Explanation of why this is suppressed
func someFunc() {
    // ...
}
```

Note that our configuration requires:
1. Specifying which linter is being disabled
2. Providing a reason for the suppression

## Configuration

The linting configuration is stored in `.golangci.yml` at the root of the project. See the [golangci-lint documentation](https://golangci-lint.run/usage/configuration/) for details on configuration options.

## Best Practices

1. Run linting locally before pushing changes.
2. Fix linting issues as they arise rather than accumulating them.
3. If you add new Go files or packages, make sure they follow our linting standards.
4. Only suppress linting issues when absolutely necessary, with clear explanations.
5. Consider using `make lint-fix` to automatically fix simple issues.