# Makefile for Terraform Package Provider

# Variables
BINARY_NAME := terraform-provider-package
VERSION := $(shell git describe --tags --always --dirty)
COMMIT_HASH := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go toolchain and build flags
GOTOOLCHAIN := go1.25.1
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH)"
GOFLAGS_BUILD := -gcflags=all=-lang=go1.25
GOFLAGS_TEST := -gcflags=all=-lang=go1.24

# Export environment variables for all targets
export GOTOOLCHAIN

# Terraform provider installation paths
TERRAFORM_PLUGINS_DIR := ~/.terraform.d/plugins
LOCAL_PROVIDER_PATH := $(TERRAFORM_PLUGINS_DIR)/jamesainslie/package/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the provider binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	GOFLAGS="$(GOFLAGS_BUILD)" go build $(LDFLAGS) -o $(BINARY_NAME) .

## install: Install the provider locally for development
.PHONY: install
install: build
	@echo "Installing provider locally..."
	mkdir -p $(LOCAL_PROVIDER_PATH)
	cp $(BINARY_NAME) $(LOCAL_PROVIDER_PATH)/

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

## test: Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	GOFLAGS="$(GOFLAGS_TEST)" go test -v ./internal/... -race

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	GOFLAGS="$(GOFLAGS_TEST)" go test -v ./internal/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-acc: Run acceptance tests (requires TF_ACC=1)
.PHONY: test-acc
test-acc:
	@echo "Running acceptance tests..."
	TF_ACC=1 GOFLAGS="$(GOFLAGS_TEST)" go test -v ./internal/provider/ -timeout=30m

## test-all: Run all tests (unit + acceptance)
.PHONY: test-all
test-all: test test-acc

## lint: Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found in PATH, trying ~/go/bin/golangci-lint..."; }
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	elif [ -f ~/go/bin/golangci-lint ]; then \
		~/go/bin/golangci-lint run ./...; \
	else \
		echo "Error: golangci-lint not found. Run 'make dev-setup' to install."; exit 1; \
	fi

## fmt: Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	gofmt -s -w .
	goimports -w .

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

## staticcheck: Run staticcheck static analysis
.PHONY: staticcheck
staticcheck:
	@echo "Running staticcheck..."
	@command -v staticcheck >/dev/null 2>&1 || { echo "staticcheck not found in PATH, trying ~/go/bin/staticcheck..."; }
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	elif [ -f ~/go/bin/staticcheck ]; then \
		~/go/bin/staticcheck ./...; \
	else \
		echo "Error: staticcheck not found. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1; \
	fi

## mod: Tidy and verify go modules
.PHONY: mod
mod:
	@echo "Tidying go modules..."
	go mod tidy
	go mod verify

## generate: Generate documentation and tools
.PHONY: generate
generate:
	@echo "Generating tools..."
	cd tools && go generate ./...
	@echo "Generating documentation..."
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	tfplugindocs generate --provider-name pkg

## release: Create a new release (requires VERSION tag)
.PHONY: release
release:
	@echo "Creating release..."
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Usage: make release VERSION=v0.1.0"; exit 1; fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)

## dev-setup: Set up development environment
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

## security: Run security checks
.PHONY: security
security:
	@echo "Running security checks..."
	govulncheck ./...
	gosec ./...

## check: Run all quality checks
.PHONY: check
check: fmt vet lint staticcheck test security
	@echo "All quality checks passed!"

## quality: Run comprehensive tests and quality checks
.PHONY: quality
quality:
	@echo "=== COMPREHENSIVE QUALITY CHECK ==="
	@echo "1. Running go vet..."
	@$(MAKE) vet
	@echo " go vet passed"
	@echo "2. Running golangci-lint..."
	@$(MAKE) lint
	@echo " golangci-lint passed"
	@echo "3. Running staticcheck..."
	@$(MAKE) staticcheck
	@echo " staticcheck passed"
	@echo "4. Running tests..."
	@$(MAKE) test
	@echo " All tests passed"
	@echo "5. Building project..."
	@GOFLAGS="$(GOFLAGS_BUILD)" go build ./...
	@echo " Build successful"
	@echo ""
	@echo " ALL QUALITY CHECKS PASSED! "

## lint-all: Run all linting tools (quick check)
.PHONY: lint-all
lint-all:
	@echo "Running all linting tools..."
	@$(MAKE) vet
	@$(MAKE) lint
	@$(MAKE) staticcheck
	@echo " All linting passed!"

## ci: Run CI-like checks locally
.PHONY: ci
ci: mod check test-coverage
	@echo "CI checks completed!"

## example-init: Initialize the test example
.PHONY: example-init
example-init: install
	@echo "Initializing test example..."
	cd examples/test-module && \
	export TF_CLI_CONFIG_FILE="$(shell pwd)/examples/test-module/.terraformrc" && \
	terraform init

## example-plan: Plan the test example
.PHONY: example-plan
example-plan: example-init
	@echo "Planning test example..."
	cd examples/test-module && \
	export TF_CLI_CONFIG_FILE="$(shell pwd)/examples/test-module/.terraformrc" && \
	terraform plan

## example-apply: Apply the test example
.PHONY: example-apply
example-apply: example-init
	@echo "Applying test example..."
	cd examples/test-module && \
	export TF_CLI_CONFIG_FILE="$(shell pwd)/examples/test-module/.terraformrc" && \
	terraform apply -auto-approve

## example-destroy: Destroy the test example
.PHONY: example-destroy
example-destroy:
	@echo "Destroying test example..."
	cd examples/test-module && \
	export TF_CLI_CONFIG_FILE="$(shell pwd)/examples/test-module/.terraformrc" && \
	terraform destroy -auto-approve

## example-clean: Clean up test example
.PHONY: example-clean
example-clean:
	@echo "Cleaning test example..."
	cd examples/test-module && rm -rf .terraform* terraform.tfstate*

# Development workflow targets
## dev: Full development workflow (format, test, build)
.PHONY: dev
dev: fmt mod vet test build
	@echo "Development workflow completed!"

## pre-commit: Run pre-commit checks
.PHONY: pre-commit
pre-commit: fmt mod vet lint test
	@echo "Pre-commit checks passed!"

# Platform-specific builds
## build-all: Build for all supported platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	mkdir -p dist/
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_amd64 .
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_arm64 .
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64 .
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_arm64 .
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_windows_amd64.exe .
	GOFLAGS="$(GOFLAGS_BUILD)" GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_freebsd_amd64 .

## docker-test: Run tests in Docker container
.PHONY: docker-test
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(shell pwd):/workspace -w /workspace golang:1.23 make test

## benchmark: Run benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	GOFLAGS="$(GOFLAGS_TEST)" go test -bench=. -benchmem ./internal/...

## profile: Run tests with profiling
.PHONY: profile
profile:
	@echo "Running tests with profiling..."
	GOFLAGS="$(GOFLAGS_TEST)" go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./internal/...
	go tool pprof cpu.prof