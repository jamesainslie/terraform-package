# Makefile for Terraform Package Provider

# Variables
BINARY_NAME := terraform-provider-pkg
VERSION := $(shell git describe --tags --always --dirty)
COMMIT_HASH := $(shell git rev-parse HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH)"

# Terraform provider installation paths
TERRAFORM_PLUGINS_DIR := ~/.terraform.d/plugins
LOCAL_PROVIDER_PATH := $(TERRAFORM_PLUGINS_DIR)/jamesainslie/pkg/$(VERSION)/$(shell go env GOOS)_$(shell go env GOARCH)

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
	go build $(LDFLAGS) -o $(BINARY_NAME) .

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
	go test -v ./internal/... -race

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v ./internal/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-acc: Run acceptance tests (requires TF_ACC=1)
.PHONY: test-acc
test-acc:
	@echo "Running acceptance tests..."
	TF_ACC=1 go test -v ./internal/provider/ -timeout=30m

## test-all: Run all tests (unit + acceptance)
.PHONY: test-all
test-all: test test-acc

## lint: Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	golangci-lint run ./...

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

## mod: Tidy and verify go modules
.PHONY: mod
mod:
	@echo "Tidying go modules..."
	go mod tidy
	go mod verify

## generate: Generate documentation
.PHONY: generate
generate:
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
check: fmt vet lint test security
	@echo "All quality checks passed!"

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
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_darwin_arm64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_linux_arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_windows_amd64.exe .
	GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)_freebsd_amd64 .

## docker-test: Run tests in Docker container
.PHONY: docker-test
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(shell pwd):/workspace -w /workspace golang:1.23 make test

## benchmark: Run benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem ./internal/...

## profile: Run tests with profiling
.PHONY: profile
profile:
	@echo "Running tests with profiling..."
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./internal/...
	go tool pprof cpu.prof