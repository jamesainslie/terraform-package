# Contributing to Terraform Package Provider

We welcome contributions to the Terraform Package Provider! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/terraform-provider-pkg.git
   cd terraform-provider-pkg
   ```
3. **Set up the development environment** (see below)

## Development Setup

### Prerequisites

- **Go 1.21+** (we use Go 1.23.7)
- **Terraform 1.5+** for testing
- **Platform-specific package managers**:
  - macOS: Homebrew (`brew`)
  - Linux: APT (`apt-get`) - Phase 3
  - Windows: winget/Chocolatey - Phase 4

### Setup Development Environment

```bash
# Install development tools
make dev-setup

# Install pre-commit hooks (recommended)
pip install pre-commit
pre-commit install

# Verify setup
make check
```

## Making Changes

### Branch Naming

Use descriptive branch names:
- `feat/add-linux-support` - New features
- `fix/homebrew-timeout-issue` - Bug fixes
- `docs/update-readme` - Documentation changes
- `test/improve-coverage` - Test improvements

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `perf`: Performance improvements

**Scopes (optional):**
- `brew`: Homebrew-related changes
- `apt`: APT-related changes
- `winget`: winget-related changes
- `core`: Core functionality
- `tests`: Test-related changes

**Examples:**
```bash
feat(brew): add support for cask pinning
fix(core): resolve timeout issues in command execution
docs: update installation instructions
test(brew): add acceptance tests for formula installation
```

## Testing

### Running Tests

```bash
# Unit tests (fast, no external dependencies)
make test

# Unit tests with coverage
make test-coverage

# Acceptance tests (requires package managers)
make test-acc

# All tests
make test-all

# Security checks
make security
```

### Test Requirements

- **Unit tests**: Required for all new functionality
- **Acceptance tests**: Required for resources and data sources
- **Coverage**: Maintain >60% overall coverage
- **Platform tests**: Test on your target platform

### Writing Tests

#### Unit Tests
```go
func TestNewFunction(t *testing.T) {
    // Test logic without external dependencies
    // Use mocks for package managers
}
```

#### Acceptance Tests
```go
func TestAccResource_Basic(t *testing.T) {
    if runtime.GOOS != "darwin" {
        t.Skip("Skipping macOS test on non-macOS platform")
    }
    
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccResourceConfig(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("pkg_package.test", "name", "hello"),
                ),
            },
        },
    })
}
```

## Submitting Changes

### Pull Request Process

1. **Create a pull request** from your fork
2. **Fill out the PR template** completely
3. **Ensure all checks pass**:
   - All tests pass
   - Code coverage maintained
   - Linting passes
   - Documentation updated
4. **Request review** from maintainers
5. **Address feedback** promptly

### PR Requirements

- [ ] **Tests**: All new functionality has tests
- [ ] **Documentation**: Updated for any user-facing changes
- [ ] **Changelog**: Entry added if user-facing change
- [ ] **Examples**: Updated if new features added
- [ ] **Backward compatibility**: Maintained unless major version

## Code Style

### Go Code Style

We follow standard Go conventions plus:

- **gofmt**: Code must be formatted with `gofmt`
- **goimports**: Imports must be organized with `goimports`
- **golangci-lint**: Must pass all linter checks
- **Comments**: Public functions and types must have comments
- **Error handling**: Comprehensive error handling required

### Terraform Code Style

- **terraform fmt**: All Terraform code must be formatted
- **Consistent naming**: Use clear, descriptive resource names
- **Documentation**: All examples must be documented

### Running Style Checks

```bash
# Format code
make fmt

# Run linters
make lint

# Check everything
make check
```

## Documentation

### Documentation Requirements

- **Provider docs**: Auto-generated from schema descriptions
- **Examples**: Working examples for all features
- **README**: Updated for new features
- **Changelog**: User-facing changes documented

### Updating Documentation

```bash
# Generate provider documentation
make generate

# Check documentation
make docs-check
```

### Documentation Style

- **Clear descriptions**: Use plain language
- **Working examples**: All examples must be tested
- **Markdown formatting**: Follow standard markdown conventions
- **Link checking**: Ensure all links work

## Development Workflow

### Typical Development Flow

1. **Create feature branch**:
   ```bash
   git checkout -b feat/new-feature
   ```

2. **Make changes with tests**:
   ```bash
   # Write code
   # Write tests
   make test
   ```

3. **Run quality checks**:
   ```bash
   make check
   ```

4. **Update documentation**:
   ```bash
   make generate
   git add docs/
   ```

5. **Commit and push**:
   ```bash
   git commit -m "feat: add new feature"
   git push origin feat/new-feature
   ```

6. **Create pull request**

### Debugging

```bash
# Debug with verbose output
TF_LOG=DEBUG make test-acc

# Profile performance
make profile

# Run specific tests
go test -v ./internal/provider/ -run TestSpecificTest
```

## Platform-Specific Development

### macOS (Homebrew) - Phase 2 ✅
- **Current status**: Fully implemented
- **Testing**: Full acceptance test suite
- **Requirements**: Homebrew installed

### Linux (APT) - Phase 3 🔄
- **Status**: Planned
- **Testing**: Ubuntu/Debian environments
- **Requirements**: APT package manager

### Windows (winget/Chocolatey) - Phase 4 🔄
- **Status**: Planned  
- **Testing**: Windows environments
- **Requirements**: winget or Chocolatey

## Getting Help

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community discussion
- **Code Review**: Request review from maintainers
- **Documentation**: Check existing docs and examples

## Recognition

Contributors will be recognized in:
- **CHANGELOG.md**: For significant contributions
- **README.md**: For major features
- **GitHub contributors**: Automatic recognition

Thank you for contributing to the Terraform Package Provider! 🎉
