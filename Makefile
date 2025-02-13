# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=ghactions-updater
DOCKER_IMAGE=ghactions-updater

.PHONY: all build test test-e2e lint clean docker_build vuln-check install-lint install-vuln-tools

# Install linting tools if not present
.PHONY: install-lint
install-lint:
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install vulnerability checking tools if not present
.PHONY: install-vuln-tools
install-vuln-tools:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	@which nancy > /dev/null || go install github.com/sonatype-nexus-community/nancy@latest

all: test test-e2e vuln-check build

# Vulnerability scanning
vuln-check: install-vuln-tools
	@echo "Running vulnerability checks..."
	govulncheck ./...
	go list -json -deps ./... | nancy sleuth

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/ghactions-updater

test:
	$(GOTEST) -v -cover ./cmd/... ./pkg/... ./tools/...

test-e2e:
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is required for e2e tests"; \
		exit 1; \
	fi
	$(GOTEST) -v -tags=e2e ./test/e2e/...

lint: install-lint
	golangci-lint run

clean:
	rm -f bin/$(BINARY_NAME)
	rm -rf vendor/

docker_build:
	docker build -t $(DOCKER_IMAGE):latest .
