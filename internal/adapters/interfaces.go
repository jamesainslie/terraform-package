// MIT License
//
// Copyright (c) 2025 Terraform Package Provider Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package adapters defines interfaces for package manager implementations.
package adapters

import (
	"context"
)

// PackageType represents the type of package
type PackageType string

const (
	PackageTypeAuto    PackageType = "auto"
	PackageTypeFormula PackageType = "formula"
	PackageTypeCask    PackageType = "cask"
)

// PackageInfo represents information about a package.
type PackageInfo struct {
	Name              string
	Version           string
	AvailableVersions []string
	Installed         bool
	Pinned            bool
	Repository        string
	Type              PackageType // Type of package (formula, cask, etc.)
}

// PackageManager defines the interface that all package manager adapters must implement.
type PackageManager interface {
	// DetectInstalled checks if a package is installed and returns its info
	DetectInstalled(ctx context.Context, name string) (*PackageInfo, error)

	// Install installs a package with optional version specification
	Install(ctx context.Context, name, version string) error

	// InstallWithType installs a package with explicit type and optional version specification
	InstallWithType(ctx context.Context, name, version string, packageType PackageType) error

	// Remove uninstalls a package
	Remove(ctx context.Context, name string) error

	// RemoveWithType uninstalls a package with explicit type
	RemoveWithType(ctx context.Context, name string, packageType PackageType) error

	// Pin pins/holds a package at current version
	Pin(ctx context.Context, name string, pin bool) error

	// UpdateCache updates the package manager's cache
	UpdateCache(ctx context.Context) error

	// Search searches for packages matching a query
	Search(ctx context.Context, query string) ([]PackageInfo, error)

	// Info retrieves detailed information about a package
	Info(ctx context.Context, name string) (*PackageInfo, error)

	// GetManagerName returns the name of the package manager
	GetManagerName() string

	// IsAvailable checks if this package manager is available on the system
	IsAvailable(ctx context.Context) bool
}

// RepositoryManager defines the interface for managing package repositories.
type RepositoryManager interface {
	// AddRepository adds a new package repository
	AddRepository(ctx context.Context, name, uri, gpgKey string) error

	// RemoveRepository removes a package repository
	RemoveRepository(ctx context.Context, name string) error

	// ListRepositories lists all configured repositories
	ListRepositories(ctx context.Context) ([]RepositoryInfo, error)
}

// RepositoryInfo represents information about a package repository.
type RepositoryInfo struct {
	Name    string
	URI     string
	GPGKey  string
	Enabled bool
}
