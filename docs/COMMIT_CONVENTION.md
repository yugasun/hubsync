# Commit Message Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages to ensure that version numbers are automatically determined based on changes.

## Commit Message Format

Each commit message consists of a **header**, an optional **body**, and an optional **footer**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

### Header

The header is mandatory and consists of:

- **type**: describes the kind of change being made
- **scope** (optional): describes what area of the codebase is being modified
- **subject**: a short description of the change

#### Types

The type is crucial for semantic versioning and must be one of the following:

- **feat**: A new feature (triggers a MINOR version bump)
- **fix**: A bug fix (triggers a PATCH version bump)
- **docs**: Documentation only changes (no version bump)
- **style**: Changes that do not affect the meaning of the code (no version bump)
- **refactor**: A code change that neither fixes a bug nor adds a feature (PATCH)
- **perf**: A code change that improves performance (PATCH)
- **test**: Adding missing tests or correcting existing tests (no version bump)
- **chore**: Changes to the build process or auxiliary tools (no version bump)

#### Breaking Changes

For breaking changes, add `!` after the type/scope or add `BREAKING CHANGE:` to the footer. This will trigger a MAJOR version bump:

```
feat!: remove deprecated API

BREAKING CHANGE: The deprecated API has been removed
```

### Examples

```
feat(syncer): add parallel processing option
```

```
fix(docker): resolve connection timeout issue
```

```
docs: update README with new API reference
```

```
feat!: completely refactor syncer module

BREAKING CHANGE: The syncer API has changed significantly
```

## Why This Matters

When you follow this convention:

1. The semantic-release bot will automatically determine the next version number
2. A changelog will be automatically generated
3. Release notes will be created for GitHub releases
4. Tags will be automatically created based on semantic versioning