SHELL := /bin/bash

BINARY_NAME := rvmm
BUILD_DIR := ./bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
TEAM_ID ?=

LDFLAGS := -s -w \
	-X github.com/rxtech-lab/rvmm/internal/buildinfo.Version=$(VERSION) \
	-X github.com/rxtech-lab/rvmm/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/rxtech-lab/rvmm/internal/buildinfo.BuildDate=$(BUILD_DATE) \
	-X github.com/rxtech-lab/rvmm/internal/buildinfo.TeamID=$(TEAM_ID)

.PHONY: deps build build-all install package clean test

deps:
	@echo "==> Fetching dependencies..."
	go mod tidy
	go mod download
	@echo "==> Dependencies fetched"

build: deps
	@echo "==> Building $(BINARY_NAME)..."
	@echo "    Version: $(VERSION)"
	@echo "    Commit:  $(COMMIT)"
	@echo "    Date:    $(BUILD_DATE)"
	mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/$(BINARY_NAME)" .
	@echo "==> Built: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: deps
	@echo "==> Building for all platforms..."
	mkdir -p $(BUILD_DIR)
	@echo "    Building darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64" .
	@echo "    Building darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64" .
	@echo "==> Built binaries:"
	ls -la "$(BUILD_DIR)/$(BINARY_NAME)"-darwin-*

install: build
	@echo "==> Installing to /usr/local/bin..."
	sudo cp "$(BUILD_DIR)/$(BINARY_NAME)" /usr/local/bin/
	@echo "==> Installed: /usr/local/bin/$(BINARY_NAME)"

package: build
	./scripts/sign.sh
	./scripts/package-notarize.sh

clean:
	@echo "==> Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)_macOS_*.pkg
	rm -f $(BINARY_NAME)_macOS_*.pkg.sha256
	@echo "==> Cleaned"

test:
	@echo "==> Running tests..."
	go test -v ./...
