# Makefile for ORC Package Management System

# Variables
BINARY_NAME := orc
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse HEAD)
LDFLAGS := -ldflags "-X github.com/jamesainslie/orc/internal/version.Version=$(VERSION) -X github.com/jamesainslie/orc/internal/version.BuildTime=$(BUILD_TIME) -X github.com/jamesainslie/orc/internal/version.Commit=$(COMMIT_HASH) -X github.com/jamesainslie/orc/internal/version.BuiltBy=make"

# Installation paths
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin
CONFIGDIR := $(PREFIX)/etc/orc
CACHEDIR := ~/.orc

# User installation paths (no sudo required)
USER_BINDIR := $(HOME)/bin
USER_CONFIGDIR := $(HOME)/.config/orc

# Artifactory configuration (matching basset)
ARTIFACTORY_URL ?= https://packageregistry.geico.net/artifactory/
ARTIFACTORY_REPO ?= titan-generic-local
ARTIFACTORY_BINARIES_RELDIR := orc/binaries

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofumpt
GOLINT := golangci-lint

# Directories
CMD_DIR := ./cmd/orc
PKG_DIR := ./pkg/...
DIST_DIR := ./dist
COVERAGE_DIR := ./coverage

# Detect OS for cross-compilation
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
    BINARY_SUFFIX :=
endif
ifeq ($(UNAME_S),Darwin)
    BINARY_SUFFIX :=
endif
ifeq ($(OS),Windows_NT)
    BINARY_SUFFIX := .exe
endif

.PHONY: all
all: clean lint test build

.PHONY: init
init: ## Initialize project dependencies
	@echo "Initializing project..."
	@$(GOMOD) download
	@$(GOMOD) tidy

.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(DIST_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) $(CMD_DIR)

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	@mkdir -p $(COVERAGE_DIR)
	@gotestsum --junitfile $(COVERAGE_DIR)/junit.xml -- \
		-v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic $(PKG_DIR)

.PHONY: test-coverage
test-coverage: test ## Generate test coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@gocov convert $(COVERAGE_DIR)/coverage.out | gocov-xml > $(COVERAGE_DIR)/coverage.xml
	@echo "Coverage report generated at $(COVERAGE_DIR)/coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem -run=^# $(PKG_DIR)

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@$(GOLINT) run --timeout 5m

.PHONY: lint-fix
lint-fix: ## Run linters with auto-fix
	@echo "Running linters with auto-fix..."
	@$(GOLINT) run --fix

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@$(GOFMT) -l -w .
	@$(GOCMD) fmt $(PKG_DIR)

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOCMD) vet $(PKG_DIR)

.PHONY: mod-update
mod-update: ## Update Go modules
	@echo "Updating Go modules..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy

.PHONY: mod-verify
mod-verify: ## Verify Go modules
	@echo "Verifying Go modules..."
	@$(GOMOD) verify

.PHONY: clean
clean: ## Clean build artifacts and cache
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@rm -rf $(COVERAGE_DIR)
	@$(GOCLEAN)
	@echo "Cleaning Go module cache..."
	@$(GOCMD) clean -modcache -cache -testcache
	@echo "Cleaning temporary files..."
	@find . -name "*.tmp" -type f -delete 2>/dev/null || true
	@find . -name ".DS_Store" -type f -delete 2>/dev/null || true
	@echo "Clean completed successfully!"

.PHONY: clean-all
clean-all: clean ## Clean everything including user data and cache
	@echo "Performing deep clean..."
	@echo "WARNING: This will remove ALL ORC data including cache and configuration!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	@echo "Removing user cache and configuration..."
	@rm -rf $(CACHEDIR)
	@rm -rf ~/.orc.yaml
	@rm -rf ~/.config/orc
	@echo "Deep clean completed!"

.PHONY: reset
reset: clean-all ## Reset ORC to initial state (clean + uninstall)
	@echo "Resetting ORC to initial state..."
	@$(MAKE) uninstall
	@echo "Reset completed!"

.PHONY: release-dry-run
release-dry-run: ## Dry run for release
	@echo "Running release dry-run..."
	@goreleaser release --snapshot --clean

.PHONY: release
release: ## Create a new release with semantic versioning
	@echo "Creating release..."
	@if [ -z "$(ARTIFACTORY_TOKEN)" ]; then \
		echo "Error: ARTIFACTORY_TOKEN environment variable is required"; \
		exit 1; \
	fi
	@if ! command -v svu >/dev/null 2>&1; then \
		echo "Installing svu for semantic versioning..."; \
		go install github.com/caarlos0/svu@latest; \
	fi
	$(eval NEXT_VERSION := $(shell svu next))
	@echo "Next version: $(NEXT_VERSION)"
	@echo "Creating and pushing tag..."
	@git tag $(NEXT_VERSION)
	@git push origin $(NEXT_VERSION)
	@echo "Tag $(NEXT_VERSION) created and pushed. GitHub Actions will handle the release."

.PHONY: release-local
release-local: ## Create a local release without pushing tags
	@echo "Creating local release..."
	@goreleaser release --snapshot --clean

.PHONY: upload-artifactory
upload-artifactory: build-all ## Upload binaries to Artifactory manually
	@echo "Uploading binaries to Artifactory..."
	@if [ -z "$(ARTIFACTORY_TOKEN)" ]; then \
		echo "Error: ARTIFACTORY_TOKEN environment variable is required"; \
		exit 1; \
	fi
	@if ! command -v jf >/dev/null 2>&1; then \
		echo "Error: JFrog CLI (jf) is required. Install from https://jfrog.com/getcli/"; \
		exit 1; \
	fi
	@mkdir -p $(DIST_DIR)/archives
	@cd $(DIST_DIR) && tar -czf archives/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(DIST_DIR) && tar -czf archives/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(DIST_DIR) && tar -czf archives/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(DIST_DIR) && tar -czf archives/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(DIST_DIR) && zip -q archives/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@for archive in $(DIST_DIR)/archives/*; do \
		echo "Uploading $$archive..."; \
		jf rt u --url "$(ARTIFACTORY_URL)" \
			--access-token "$(ARTIFACTORY_TOKEN)" \
			"$$archive" \
			"$(ARTIFACTORY_REPO)/$(ARTIFACTORY_BINARIES_RELDIR)/$(VERSION)/$$(basename $$archive)"; \
	done

.PHONY: install
install: build ## Install ORC for current user (no sudo required)
	@echo "Installing $(BINARY_NAME) for current user..."
	@mkdir -p $(USER_BINDIR)
	@cp $(DIST_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) $(USER_BINDIR)/
	@chmod +x $(USER_BINDIR)/$(BINARY_NAME)$(BINARY_SUFFIX)
	@echo "Creating user configuration directories..."
	@mkdir -p $(USER_CONFIGDIR)
	@mkdir -p $(CACHEDIR)
	@echo "Installation completed successfully!"
	@echo ""
	@echo "ORC has been installed to: $(USER_BINDIR)/$(BINARY_NAME)"
	@echo "Configuration directory: $(USER_CONFIGDIR)"
	@echo "Cache directory: $(CACHEDIR)"
	@echo ""
	@echo "Add $(USER_BINDIR) to your PATH if not already present:"
	@echo "  echo 'export PATH=\$$HOME/bin:\$$PATH' >> ~/.bashrc"
	@echo "  echo 'export PATH=\$$HOME/bin:\$$PATH' >> ~/.zshrc"
	@echo ""
	@echo "Run 'orc --help' to get started"
	@echo "Run 'orc cache stats' to check cache status"

.PHONY: install-system
install-system: build ## Install ORC system-wide (requires sudo)
	@echo "Installing $(BINARY_NAME) system-wide to $(BINDIR)..."
	@echo "This requires sudo privileges..."
	@sudo mkdir -p $(BINDIR)
	@sudo cp $(DIST_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX) $(BINDIR)/
	@sudo chmod +x $(BINDIR)/$(BINARY_NAME)$(BINARY_SUFFIX)
	@echo "Creating system configuration directories..."
	@sudo mkdir -p $(CONFIGDIR)
	@mkdir -p $(CACHEDIR)
	@echo "System installation completed successfully!"
	@echo ""
	@echo "ORC has been installed to: $(BINDIR)/$(BINARY_NAME)"
	@echo "Configuration directory: $(CONFIGDIR)"
	@echo "Cache directory: $(CACHEDIR)"
	@echo ""
	@echo "Run 'orc --help' to get started"
	@echo "Run 'orc cache stats' to check cache status"

.PHONY: uninstall
uninstall: ## Uninstall ORC from user directory (no sudo required)
	@echo "Uninstalling $(BINARY_NAME) from user directory..."
	@rm -f $(USER_BINDIR)/$(BINARY_NAME)$(BINARY_SUFFIX)
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)$(BINARY_SUFFIX)
	@echo "Removing user configuration directories..."
	@rm -rf $(USER_CONFIGDIR)
	@echo "Cleaning user cache directory..."
	@rm -rf $(CACHEDIR)/cache.db*
	@rm -rf $(CACHEDIR)/state
	@echo ""
	@echo "ORC has been uninstalled from user directory!"
	@echo "Note: Cache directory $(CACHEDIR) has been cleaned but not removed"

.PHONY: uninstall-system
uninstall-system: ## Uninstall ORC from system directories (requires sudo)
	@echo "Uninstalling $(BINARY_NAME) from system directories..."
	@echo "This requires sudo privileges..."
	@sudo rm -f $(BINDIR)/$(BINARY_NAME)$(BINARY_SUFFIX)
	@echo "Removing system configuration directories..."
	@sudo rm -rf $(CONFIGDIR)
	@echo "Cleaning user cache directory..."
	@rm -rf $(CACHEDIR)/cache.db*
	@rm -rf $(CACHEDIR)/state
	@echo ""
	@echo "ORC has been uninstalled from system directories!"
	@echo "Note: User cache directory $(CACHEDIR) has been cleaned but not removed"

.PHONY: run
run: build ## Build and run the binary
	@echo "Running $(BINARY_NAME)..."
	@$(DIST_DIR)/$(BINARY_NAME)$(BINARY_SUFFIX)

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "Watching for changes..."
	@air -c .air.toml

.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@go doc -all ./... > docs/API.md

# Pre-commit hooks setup
.PHONY: setup-precommit
setup-precommit: ## Set up pre-commit hooks
	@echo "Setting up pre-commit hooks..."
	@if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "Installing pre-commit..."; \
		pip install pre-commit || brew install pre-commit; \
	fi
	@pre-commit install
	@pre-commit install --hook-type commit-msg
	@echo "Pre-commit hooks installed successfully"

.PHONY: run-precommit
run-precommit: ## Run pre-commit hooks on all files
	@echo "Running pre-commit hooks on all files..."
	@pre-commit run --all-files

.PHONY: update-precommit
update-precommit: ## Update pre-commit hooks to latest versions
	@echo "Updating pre-commit hooks..."
	@pre-commit autoupdate

.PHONY: help
help: ## Display this help message
	@echo "ORC Makefile Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
