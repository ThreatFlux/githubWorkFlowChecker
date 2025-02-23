# Required versions
REQUIRED_GO_VERSION = 1.24.0

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

# Coverage output paths
COVERAGE_PROFILE = coverage.out
COVERAGE_HTML = coverage.html

# Binary information
BINARY_NAME = ghactions-updater
BINARY_PATH = bin/$(BINARY_NAME)

.PHONY: all build test e2e lint clean docker-build check-versions install-tools security help version-info coverage

# Version check targets
.PHONY: check-versions
check-versions: ## Check all required tool versions
	@echo "Checking required tool versions..."
	@echo "Checking Go version..."
	@$(GO) version
	@$(GO) version | grep -q "go$(REQUIRED_GO_VERSION)" || (echo "Error: Required Go version $(REQUIRED_GO_VERSION) not found" && exit 1)
	@echo "Go version check passed"
	
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
	@echo "Building from: .pkg/cmd/ghactions-updater"
	@echo "Output binary: $(BINARY_PATH)"
	cd pkg/cmd/ghactions-updater && $(GO) build $(BUILD_FLAGS) -o ../../../$(BINARY_PATH)  || (echo "Build failed. See error above." && exit 1)
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
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required for e2e tests"; \
		exit 1; \
	fi
	@$(GO) test $(TEST_FLAGS)  ./pkg/...

coverage: ## Generate test coverage report and HTML output
	@echo "Generating coverage report..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required for coverage"; \
		exit 1; \
	fi
	@$(GO) test -coverprofile=$(COVERAGE_PROFILE) ./pkg/...
	@$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage profile written to: $(COVERAGE_PROFILE)"
	@echo "HTML coverage report written to: $(COVERAGE_HTML)"
	@echo "Coverage summary:"
	@$(GO) tool cover -func=$(COVERAGE_PROFILE)

security: install-tools ## Run security scans
	@echo "Running security scans..."
	@echo "Running gosec..."
	@$(GOSEC) ./... || (echo "Security scan failed. See errors above." && exit 1)
	@echo "Running govulncheck for dependency scanning..."
	@$(GOVULNCHECK) ./... || (echo "Dependency security scan failed. See errors above." && exit 1)
	@echo "Running nancy for dependency scanning..."
	@go list -json -deps ./... | nancy sleuth || (echo "Nancy security scan failed. See errors above." && exit 1)

clean: ## Remove build artifacts, test cache, and generated files
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