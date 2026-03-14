# CI/CD Pipeline Configuration

This document describes the CI/CD pipeline for BlockchainVPN.

## Overview

BlockchainVPN uses GitHub Actions for continuous integration and deployment.

## Workflow Files

### CI Pipeline (`.github/workflows/ci.yml`)

Runs on every push and pull request.

**Jobs:**
1. `unit-and-build` - Ubuntu latest
   - Checkout code
   - Setup Go 1.25.x
   - Run unit tests (`make test`)
   - Build CLI for all platforms (`make build-cli-all`)
   - Build GUI (native)

**Trigger Conditions:**
- On push to any branch
- On pull request to any branch

### Release Pipeline (`.github/workflows/release.yml`)

Runs when a version tag is pushed.

**Jobs:**
1. Build and release binaries for Linux, macOS, Windows

## Build Targets

The Makefile provides these build targets:

| Target | Description |
|--------|-------------|
| `make build` | Build CLI for current platform |
| `make build-gui` | Build GUI (deprecated) |
| `make build-cli-all` | Cross-compile CLI for all platforms |
| `make build-linux` | Build for Linux AMD64 |
| `make build-darwin` | Build for macOS AMD64 |
| `make build-windows` | Build for Windows AMD64 |

## Testing

| Target | Description |
|--------|-------------|
| `make test` | Run unit tests |
| `make test-unit` | Run unit tests (explicit) |
| `make test-functional` | Run functional tests |

## Local Development

Run the full CI locally:

```bash
# Run tests
make test

# Build all targets
make build-cli-all

# Format code
make fmt

# Clean up
make clean
```

## Deployment Pipeline

When a version tag is pushed (e.g., `v1.2.3`):

1. CI runs tests and builds
2. Release workflow triggers
3. Binaries are built for all platforms
4. GitHub Release is created automatically
5. Binaries are attached to release

## Version Management

Version is managed in these files:
- `VERSION` - Simple version file
- `internal/version/version.go` - Version constant

Use `make bump-version` to increment:
```bash
# Increment patch version
make bump-version

# Set specific version
make bump-version VERSION_OVERRIDE=1.2.3
```

## Secrets

Required secrets for release:
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

## Go Version

Minimum Go version: 1.25.x

Specified in:
- `.github/workflows/ci.yml`
- `go.mod`
