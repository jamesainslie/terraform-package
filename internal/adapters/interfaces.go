// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package adapters

import (
	"context"
)

// PackageInfo represents information about a package.
type PackageInfo struct {
	Name              string
	Version           string
	AvailableVersions []string
	Installed         bool
	Pinned            bool
	Repository        string
}

// PackageManager defines the interface that all package manager adapters must implement.
type PackageManager interface {
	// DetectInstalled checks if a package is installed and returns its info
	DetectInstalled(ctx context.Context, name string) (*PackageInfo, error)

	// Install installs a package with optional version specification
	Install(ctx context.Context, name, version string) error

	// Remove uninstalls a package
	Remove(ctx context.Context, name string) error

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
