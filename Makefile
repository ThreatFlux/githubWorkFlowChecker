# Required versions
REQUIRED_GO_VERSION = 1.24.0
REQUIRED_DOCKER_VERSION = 27.5.1

# Tool paths and versions
GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOSEC ?= gosec
GOVULNCHECK ?= govulncheck
DOCKER ?= docker

# Build flags
BUILD_FLAGS ?= -v
TEST_FLAGS ?= -v -race -cover
LINT_FLAGS ?= run --timeout=5m

# Binary information
BINARY_NAME = ghactions-updater
BINARY_PATH = bin/$(BINARY_NAME)

.PHONY: all build test e2e lint clean docker-build check-versions install-tools security help version-info

# Version check targets
.PHONY: check-versions
check-versions: ## Check all required tool versions
	@echo "Checking required tool versions..."
	@echo "Checking Go version..."
	@$(GO) version
	@$(GO) version | grep -q "go$(REQUIRED_GO_VERSION)" || (echo "Error: Required Go version $(REQUIRED_GO_VERSION) not found" && exit 1)
	@echo "Go version check passed"
	
	@echo "Checking Docker version..."
	@$(DOCKER) --version
	@$(DOCKER) --version | grep -q "$(REQUIRED_DOCKER_VERSION)" || (echo "Error: Required Docker version $(REQUIRED_DOCKER_VERSION) not found" && exit 1)
	@echo "Docker version check passed"
	
	@echo "All required versions found"

# Install required tools
install-tools: ## Install required Go tools
	@echo "Installing security and linting tools..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/sonatype-nexus-community/nancy@latest

build: check-versions ## Build the application
	@echo "Building application..."
	@mkdir -p bin
	@echo "Running go build with flags: $(BUILD_FLAGS)"
	@echo "Building from: ./cmd/ghactions-updater"
	@echo "Output binary: $(BINARY_PATH)"
	$(GO) build $(BUILD_FLAGS) -o $(BINARY_PATH) ./cmd/ghactions-updater || (echo "Build failed. See error above." && exit 1)
	@if [ -f "$(BINARY_PATH)" ]; then \
		echo "Build successful. Binary created at $(BINARY_PATH)"; \
	else \
		echo "Build failed. Binary not created."; \
		exit 1; \
	fi

lint: install-tools ## Run golangci-lint for code analysis
	@echo "Running linters..."
	@echo "Using golangci-lint with flags: $(LINT_FLAGS)"
	$(GOLANGCI_LINT) $(LINT_FLAGS) ./... || (echo "Linting failed. See errors above." && exit 1)

test: ## Run unit tests with coverage
	@echo "Running tests..."
	@$(GO) test $(TEST_FLAGS) ./cmd/... ./pkg/... ./tools/...

e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required for e2e tests"; \
		exit 1; \
	fi
	@$(GO) test $(TEST_FLAGS) -tags=e2e ./test/e2e/...

security: install-tools ## Run security scans
	@echo "Running security scans..."
	@echo "Running gosec..."
	@$(GOSEC) ./... || (echo "Security scan failed. See errors above." && exit 1)
	@echo "Running govulncheck for dependency scanning..."
	@$(GOVULNCHECK) ./... || (echo "Dependency security scan failed. See errors above." && exit 1)
	@echo "Running nancy for dependency scanning..."
	@go list -json -deps ./... | nancy sleuth || (echo "Nancy security scan failed. See errors above." && exit 1)

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_PATH)
	@rm -rf vendor/
	@go clean -testcache

docker-build: check-versions ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

all: test e2e security build ## Run all checks and build

help: ## Display available commands
	@echo "Available commands:"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

version-info: ## Display version information
	@echo "Required Versions:"
	@echo "  Go:     $(REQUIRED_GO_VERSION)+"
	@echo "  Docker: $(REQUIRED_DOCKER_VERSION)+"
	@echo "\nInstalled Versions:"
	@$(GO) version
	@$(DOCKER) --version
