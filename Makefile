# Go parameters
BINARY_NAME=ghactions-updater
MAIN_PACKAGE=./cmd/ghactions-updater
GO=go
DOCKER_IMAGE=ghactions-updater
DOCKER_TAG=latest

# Build the binary
.PHONY: build
build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

# Run tests with coverage
.PHONY: test
test:
	$(GO) test -v -cover ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run ./...

# Build Docker image
.PHONY: dockerbuild
dockerbuild:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/
	$(GO) clean

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod download
	$(GO) mod verify

# Run all checks (lint and test)
.PHONY: check
check: lint test

# Default target
.DEFAULT_GOAL := build
