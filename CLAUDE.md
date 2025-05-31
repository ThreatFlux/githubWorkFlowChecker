# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based security tool called **GitHub Actions Workflow Checker** that automatically updates GitHub Actions workflows to use pinned commit SHAs instead of floating tags, protecting against supply chain attacks. The tool can run as a CLI application or as a GitHub Action.

## Essential Commands

### Development Commands
- `make build` - Build the binary (`bin/ghactions-updater`)
- `make test` - Run unit tests (requires GITHUB_TOKEN environment variable)
- `make lint` - Run golangci-lint for code analysis
- `make security` - Run security scans (gosec, govulncheck, nancy)
- `make fmt` - Format Go source files
- `make coverage` - Generate test coverage report
- `make all` - Run all checks and build

### Testing
- `make test` - Run all tests (requires GITHUB_TOKEN)
- `go test ./pkg/...` - Run tests directly
- `go test -run TestSpecificFunction ./pkg/updater/` - Run specific test

### Docker Development
- `make docker-build` - Build Docker image
- `make docker-test` - Test Docker image
- `make docker-dev-build` - Build development environment
- `make docker-tests` - Run tests in Docker

### Installation and Dependencies
- `make install-tools` - Install required Go tools (gosec, govulncheck, golangci-lint, etc.)
- `go mod download` - Download dependencies

## Architecture

### Core Components

**Main Entry Point** (`pkg/cmd/ghactions-updater/main.go`):
- CLI application with flags for repo path, owner, repo name, token, workflows path
- Supports dry-run mode, staging mode, and normal PR creation mode
- Uses factory pattern for dependency injection (version checker, PR creator)

**Core Packages**:

1. **Scanner** (`pkg/updater/scanner.go`):
   - Scans `.github/workflows` directory for YAML workflow files
   - Parses action references from workflow files
   - Extracts owner, name, version, commit hash, and line information

2. **Version Checker** (`pkg/updater/version_checker.go`):
   - Interface-based design for checking GitHub Action versions
   - Uses GitHub API to fetch latest versions and commit hashes
   - Compares current vs latest versions to determine update availability

3. **Update Manager** (`pkg/updater/update_manager.go`):
   - Creates update objects containing old/new version information
   - Applies updates to workflow files while preserving comments
   - Handles commit hash updates alongside version updates

4. **PR Creator** (`pkg/updater/pr_creator.go`):
   - Creates GitHub pull requests with action updates
   - Generates detailed PR descriptions with security benefits
   - Configurable labels and commit messages

**Key Interfaces** (`pkg/updater/interfaces.go`):
- `VersionChecker` - For checking action versions
- `PRCreator` - For creating pull requests  
- `UpdateManager` - For managing updates

**Common Utilities** (`pkg/common/`):
- File utilities, GitHub utilities, path utilities, string utilities
- Centralized error handling and constants

### Data Flow

1. **Scan**: Scanner finds workflow files and parses action references
2. **Check**: Version checker determines which actions need updates
3. **Create**: Update manager creates update objects with new versions/hashes
4. **Apply**: Either create PR (normal), apply locally (stage), or preview (dry-run)

### Testing Strategy

The codebase uses comprehensive testing with:
- Unit tests for each component (`*_test.go` files)
- Integration tests (`pkg/test/e2e/`)
- Test utilities and mock servers (`pkg/common/testutils/`)
- Coverage reporting via `make coverage`

**Test Requirements**: 
- GITHUB_TOKEN environment variable required for tests that interact with GitHub API
- Tests use table-driven test patterns extensively
- Mock servers for testing GitHub API interactions

### Security Focus

- All dependencies are pinned and regularly scanned
- Docker images run with minimal privileges (`--cap-drop=ALL`)
- Tool specifically designed to improve supply chain security
- Security scans integrated into build process (`make security`)