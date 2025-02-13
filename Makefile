# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
BINARY_NAME=ghactions-updater
DOCKER_IMAGE=ghactions-updater

.PHONY: all build test lint clean dockerbuild

all: test build

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) ./cmd/ghactions-updater

test:
	$(GOTEST) -v -cover ./...

lint:
	golangci-lint run

clean:
	rm -f bin/$(BINARY_NAME)
	rm -rf vendor/

dockerbuild:
	docker build -t $(DOCKER_IMAGE):latest .
