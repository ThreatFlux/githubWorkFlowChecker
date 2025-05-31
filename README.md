# GitHub Actions Workflow Checker
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/ThreatFlux/githubWorkFlowChecker)](https://github.com/ThreatFlux/githubWorkFlowChecker/releases)
[![CI](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/ci.yml/badge.svg)](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/ci.yml)
[![Release](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/release.yml/badge.svg)](https://github.com/ThreatFlux/githubWorkFlowChecker/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/ThreatFlux/githubWorkFlowChecker/branch/main/graph/badge.svg)](https://codecov.io/gh/ThreatFlux/githubWorkFlowChecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/ThreatFlux/githubWorkFlowChecker)](https://goreportcard.com/report/github.com/ThreatFlux/githubWorkFlowChecker)
[![GoDoc](https://godoc.org/github.com/ThreatFlux/githubWorkFlowChecker?status.svg)](https://godoc.org/github.com/ThreatFlux/githubWorkFlowChecker)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=ThreatFlux_githubWorkFlowChecker&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=ThreatFlux_githubWorkFlowChecker)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


A security-focused tool that automatically updates GitHub Actions workflows to use pinned commit SHAs instead of floating tags, protecting against supply chain attacks while maintaining compatibility.

## 🔐 Security Features

- Automatically updates GitHub Actions to use pinned commit SHAs
- Prevents supply chain attacks by ensuring verified action versions
- Maintains workflow compatibility through testing
- Creates automated pull requests with security improvements
- Includes version information alongside hash updates

## ✨ Key Features

- Scans GitHub Actions workflow files (`.yml` and `.yaml`)
- Creates pull requests with detailed security improvements
- Supports both CLI and GitHub Actions workflow usage
- Handles semantic versioning and commit SHA references
- Runs in a secure Docker container with minimal permissions
- Provides detailed security reports

## 🚀 Quick Start

### GitHub Actions Workflow (Recommended)

Add this workflow to your repository:

```yaml
name: Update GitHub Actions Dependencies

on:
  schedule:
    - cron: "0 0 * * 1"  # Runs every Monday
  workflow_dispatch:      # Manual trigger option
    inputs:
      dry-run:
        description: 'Show changes without applying them'
        required: false
        default: 'false'
        type: boolean
      workflows-path:
        description: 'Path to workflow files'
        required: false
        default: '.github/workflows'
        type: string

jobs:
  update-actions:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2
      - name: Update GitHub Actions
        uses: ThreatFlux/githubWorkFlowChecker@fc3d69cb98fb60b80a6009169959831d4f49ee7d  # v1.20250309.1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          owner: ${{ github.repository_owner }}
          repo-name: ${{ github.event.repository.name }}
          labels: "dependencies,security"
          # Optional parameters
          workflows-path: ${{ inputs.workflows-path }}
          dry-run: ${{ inputs.dry-run }}
          # stage: 'false'  # Uncomment to apply changes locally without creating a PR
```

### CLI Installation


#### Using Docker
```bash
docker pull ghcr.io/threatflux/ghactions-updater:latest
```

## 📋 Usage

### CLI Options

```bash
ghactions-updater [options]
```

| Option | Description | Required | Default |
|--------|-------------|----------|---------|
| `-token` | GitHub token with PR permissions | ✅ | - |
| `-owner` | Repository owner | ✅ | - |
| `-repo-name` | Repository name | ✅ | - |
| `-repo` | Repository path | ❌ | "." |
| `-workflows-path` | Path to workflow files | ❌ | ".github/workflows" |
| `-dry-run` | Show changes without applying them | ❌ | false |
| `-stage` | Apply changes locally without creating PR | ❌ | false |
| `-version` | Print version information | ❌ | - |

### Environment Variables

- `GITHUB_TOKEN`: Alternative to `-token` flag
- `OWNER`: Alternative to `-owner` flag
- `REPO_NAME`: Alternative to `-repo-name` flag
- `WORKFLOWS_PATH`: Alternative to `-workflows-path` flag

## 🛠️ Development

### Prerequisites

- Go 1.24.2 or later
- Make
- Docker (optional)
- Git

### Local Setup

1. Clone the repository:
```bash
git clone https://github.com/ThreatFlux/githubWorkFlowChecker.git
cd githubWorkFlowChecker
```

2. Install dependencies:
```bash
make install-tools
go mod download
```

### Common Tasks

| Command | Description |
|---------|-------------|
| `make build` | Build binary |
| `make test` | Run tests |
| `make lint` | Run linter |
| `make security` | Run security checks |
| `make docker-build` | Build Docker image |
| `make clean` | Clean up build artifacts |

## 📚 Documentation

- [Security Policy](SECURITY.md) - Security policy and reporting vulnerabilities
- [Contributing Guidelines](CONTRIBUTING.md) - Guidelines for contributing
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community behavior guidelines

## 🔒 Security

- All dependencies are regularly updated and scanned for vulnerabilities
- Docker images are signed and include SBOMs
- Actions are pinned to specific commit SHAs
- Minimal container permissions and secure defaults

Report security vulnerabilities via [GitHub Security Advisories](https://github.com/ThreatFlux/githubWorkFlowChecker/security/advisories/new)

## 📜 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) before submitting a pull request.

## 📬 Support

- Open an [issue](https://github.com/ThreatFlux/githubWorkFlowChecker/issues)
- Start a [discussion](https://github.com/ThreatFlux/githubWorkFlowChecker/discussions)
- Email: wyattroersma@gmail.com

## ⭐ Acknowledgments

Thanks to all contributors and the GitHub Actions community for making this tool possible.
