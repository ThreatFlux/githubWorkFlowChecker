# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=ghactions-updater
DOCKER_IMAGE=ghactions-updater

.PHONY: all build test test-e2e lint clean dockerbuild

all: test test-e2e build

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

lint:
	golangci-lint run

clean:
	rm -f bin/$(BINARY_NAME)
	rm -rf vendor/

dockerbuild:
	docker build -t $(DOCKER_IMAGE):latest .
