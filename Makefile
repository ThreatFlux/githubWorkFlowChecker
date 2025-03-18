# Required versions
REQUIRED_GO_VERSION = 1.24.1
REQUIRED_DOCKER_VERSION = 24.0.0

# Tool paths and versions
GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOSEC ?= gosec
GOVULNCHECK ?= govulncheck
DOCKER ?= docker
COSIGN ?= cosign
SYFT ?= syft

# Version information
VERSION ?= $(shell git describe --tags --always)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Build flags
BUILD_FLAGS ?= -v
TEST_FLAGS ?= -v -race -cover
LINT_FLAGS ?= run --timeout=5m

# Coverage output paths
COVERAGE_PROFILE = coverage.out
COVERAGE_HTML = coverage.html

# Binary information
BINARY_NAME = ghactions-updater
BINARY_PATH = bin/$(BINARY_NAME)

# Docker information
DOCKER_REGISTRY ?= threatflux
DOCKER_IMAGE = $(DOCKER_REGISTRY)/$(BINARY_NAME)
DOCKER_TAG ?= $(VERSION)
DOCKER_LATEST = $(DOCKER_IMAGE):latest
DOCKER_DEV_IMAGE = $(DOCKER_REGISTRY)/go-dev

.PHONY: all build test lint clean docker-build check-versions install-tools security help version-info coverage docker-push docker-sign docker-verify install docker-run fmt docker-test docker-tests docker-dev-build docker-fmt docker-lint docker-security docker-coverage docker-all docker-shell

# Version check targets
check-versions: ## Check all required tool versions
	@echo "Checking required tool versions..."
	@echo "Checking Go version..."
	@$(GO) version | grep -q "go$(REQUIRED_GO_VERSION)" || (echo "Error: Required Go version $(REQUIRED_GO_VERSION) not found" && exit 1)
	@echo "Checking Docker version..."
@$(DOCKER) --version | grep -q "$(REQUIRED_DOCKER_VERSION)" || (echo "Warning: Recommended Docker version $(REQUIRED_DOCKER_VERSION) not found")
	@echo "All version checks completed"

# Install required tools
install-tools: ## Install required Go tools
	@echo "Installing security and linting tools..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/sonatype-nexus-community/nancy@latest
	@go install github.com/sigstore/cosign/cmd/cosign@latest
	@go install github.com/anchore/syft/cmd/syft@latest

build: check-versions ## Build the application
	@echo "Building application..."
	@mkdir -p bin
	cd pkg/cmd/$(BINARY_NAME)/ && $(GO) build $(BUILD_FLAGS) \
		-ldflags="-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)" \
		-o ../../../$(BINARY_PATH)

fmt: ## Format Go source files
	@echo "Formatting Go files..."
	@find pkg -name "*.go" -type f -exec $(GO) fmt {} \;

lint: install-tools ## Run golangci-lint for code analysis
	@echo "Running linters..."
	$(GOLANGCI_LINT) $(LINT_FLAGS) ./...

test: ## Run unit tests with coverage
	@echo "Running tests..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required for tests"; \
		exit 1; \
	fi
	@$(GO) test $(TEST_FLAGS) ./pkg/...

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required"; \
		exit 1; \
	fi
	@$(GO) test -coverprofile=$(COVERAGE_PROFILE) ./pkg/...
	@$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@$(GO) tool cover -func=$(COVERAGE_PROFILE)

security: install-tools ## Run security scans
	@echo "Running security scans..."
	@$(GOSEC) ./...
	@$(GOVULNCHECK) ./...
	@go list -json -deps ./... | nancy sleuth

docker-build: check-versions ## Build Docker image
	@echo "Building Docker image..."
	@$(DOCKER) build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_LATEST) \
		.

docker-sign: ## Sign Docker image with cosign
	@echo "Signing Docker image..."
	@$(COSIGN) sign --key cosign.key $(DOCKER_IMAGE):$(DOCKER_TAG)
	@$(COSIGN) sign --key cosign.key $(DOCKER_LATEST)

docker-test: ## Test Docker image with
	@echo "Testing Docker image..."
	@$(DOCKER) run \
		--cap-drop=ALL \
		-e GITHUB_TOKEN \
		$(DOCKER_IMAGE):$(DOCKER_TAG) -h

docker-verify: ## Verify Docker image signature
	@echo "Verifying Docker image signature..."
	@$(COSIGN) verify --key cosign.pub $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-run: ## Run Docker container with security options
	@echo "Running Docker container with security options..."
	@$(DOCKER) run \
		--cap-drop=ALL \
		-e GITHUB_TOKEN \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: docker-build docker-sign ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@$(DOCKER) push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@$(DOCKER) push $(DOCKER_LATEST)

install: build ## Install the binary
	@echo "Installing $(BINARY_NAME)..."
	@install -m 755 $(BINARY_PATH) /usr/local/bin/$(BINARY_NAME)

clean: ## Remove build artifacts and generated files
	@echo "Cleaning all artifacts and generated files..."
	@rm -f $(BINARY_PATH)
	@rm -f $(COVERAGE_PROFILE)
	@rm -f $(COVERAGE_HTML)
	@rm -rf vendor/
	@rm -rf bin/
	@rm -f *.log
	@rm -f *.out
	@rm -f *.test
	@rm -f *.prof
	@rm -rf dist/
	@go clean -cache -testcache -modcache -fuzzcache

all: fmt test security lint build docker-build ## Run all checks and build

help: ## Display available commands
	@echo "Available commands:"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

version-info: ## Display version information
	@echo "Build Information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build Date: $(BUILD_DATE)"
	@echo "\nRequired Versions:"
	@echo "  Go:     $(REQUIRED_GO_VERSION)+"
	@echo "  Docker: $(REQUIRED_DOCKER_VERSION)+"
	@echo "\nInstalled Versions:"
	@$(GO) version
	@$(DOCKER) --version

# Docker development environment targets
docker-dev-build: ## Build the development Docker image
	@echo "Building development Docker image..."
	@$(DOCKER) build -t $(DOCKER_DEV_IMAGE) -f Dockerfile.dev .

docker-fmt: docker-dev-build ## Format Go source files using Docker
	@echo "Formatting Go files using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace $(DOCKER_DEV_IMAGE) fmt

docker-lint: docker-dev-build ## Run golangci-lint for code analysis using Docker
	@echo "Running linters using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace $(DOCKER_DEV_IMAGE) lint

docker-security: docker-dev-build ## Run security scans using Docker
	@echo "Running security scans using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace $(DOCKER_DEV_IMAGE) security

docker-tests: docker-dev-build ## Run unit tests with coverage using Docker
	@echo "Running tests using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace -e GITHUB_TOKEN=$(GITHUB_TOKEN) $(DOCKER_DEV_IMAGE) test

docker-coverage: docker-dev-build ## Generate test coverage report using Docker
	@echo "Generating coverage report using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace -e GITHUB_TOKEN=$(GITHUB_TOKEN) $(DOCKER_DEV_IMAGE) coverage

docker-all: docker-dev-build ## Run all checks and tests using Docker
	@echo "Running all checks and tests using Docker..."
	@$(DOCKER) run -v $(CURDIR):/workspace -e GITHUB_TOKEN=$(GITHUB_TOKEN) $(DOCKER_DEV_IMAGE) all

docker-shell: docker-dev-build ## Start a shell in the development container
	@echo "Starting shell in development container..."
	@$(DOCKER) run -it -v $(CURDIR):/workspace $(DOCKER_DEV_IMAGE) shell