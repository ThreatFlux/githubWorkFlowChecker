# GitHub Actions Workflow Checker

[![CI](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/ci.yml/badge.svg)](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/ci.yml)
[![Release](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/release.yml/badge.svg)](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/ThreatFlux/githubWorkFlowChecker/branch/main/graph/badge.svg)](https://codecov.io/gh/ThreatFlux/githubWorkFlowChecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/ThreatFlux/githubWorkFlowChecker)](https://goreportcard.com/report/github.com/ThreatFlux/githubWorkFlowChecker)
[![GoDoc](https://godoc.org/github.com/ThreatFlux/githubWorkFlowChecker?status.svg)](https://godoc.org/github.com/ThreatFlux/githubWorkFlowChecker)

A tool to automatically check and update GitHub Actions workflow dependencies. It scans workflow files for action references, checks for newer versions, and creates pull requests with updates.

## Features

- Scans GitHub Actions workflow files (`.yml` and `.yaml`)
- Checks for newer versions of actions using GitHub API
- Creates pull requests with updates
- Supports both CLI usage and GitHub Actions workflow
- Handles semantic versioning and commit SHA references
- Runs in Docker container

## Installation

### Using Go

```bash
go install github.com/ThreatFlux/githubWorkFlowChecker/cmd/ghactions-updater@latest
```

### Using Docker

```bash
docker pull ghcr.io/threatflux/ghactions-updater:latest
```

## Usage

### CLI

```bash
ghactions-updater -owner <owner> -repo-name <repo> -token <github-token>
```

Options:
- `-owner`: Repository owner (required)
- `-repo-name`: Repository name (required)
- `-token`: GitHub token (required, can also be set via GITHUB_TOKEN environment variable)
- `-repo`: Path to repository (default: ".")
- `-version`: Print version information and exit

Example:
```bash
# Check version
ghactions-updater -version

# Update workflows
ghactions-updater -owner <owner> -repo-name <repo> -token <github-token>
```

### GitHub Actions Workflow

```yaml
name: Update Actions

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker://ghcr.io/threatflux/ghactions-updater:latest
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          OWNER: ${{ github.repository_owner }}
          REPO_NAME: ${{ github.event.repository.name }}
```

## Development

### Requirements

- Go 1.24.0 or later
- Make
- Docker (optional)

### Setup

1. Clone the repository
```bash
git clone https://github.com/ThreatFlux/githubWorkFlowChecker.git
cd githubWorkFlowChecker
```

2. Install dependencies
```bash
go mod download
```

### Common Tasks

- Build binary: `make build`
- Run tests: `make test`
- Run linter: `make lint`
- Build Docker image: `make docker_build`
- Clean up: `make clean`

## Documentation

- [API Documentation](docs/api.md) - Detailed documentation of the package API, interfaces, and best practices
- [Contributing Guidelines](CONTRIBUTING.md) - Guidelines for contributing to the project

## License

MIT License - see LICENSE file for details.
