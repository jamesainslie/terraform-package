// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package registry

import (
	"context"
	"runtime"
)

// PackageMapping represents cross-platform package name mappings.
type PackageMapping struct {
	LogicalName string
	Darwin      string // macOS/Homebrew
	Linux       string // Linux/APT
	Windows     string // Windows/winget
}

// PackageRegistry defines the interface for package name resolution.
type PackageRegistry interface {
	// ResolvePackageName resolves a logical package name to platform-specific name
	ResolvePackageName(ctx context.Context, logicalName string, platform string) (string, error)

	// GetPackageMapping retrieves the full mapping for a logical package name
	GetPackageMapping(ctx context.Context, logicalName string) (*PackageMapping, error)

	// ListPackages returns all available package mappings
	ListPackages(ctx context.Context) ([]PackageMapping, error)

	// AddPackageMapping adds or updates a package mapping
	AddPackageMapping(ctx context.Context, mapping PackageMapping) error
}

// DefaultRegistry provides a default implementation with embedded mappings.
type DefaultRegistry struct {
	mappings map[string]PackageMapping
}

// NewDefaultRegistry creates a new default registry with common package mappings.
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		mappings: getDefaultMappings(),
	}
}

// ResolvePackageName resolves a logical package name for the current platform.
func (r *DefaultRegistry) ResolvePackageName(ctx context.Context, logicalName string, platform string) (string, error) {
	if platform == "" {
		platform = runtime.GOOS
	}

	mapping, exists := r.mappings[logicalName]
	if !exists {
		// If no mapping exists, return the logical name as-is
		return logicalName, nil
	}

	switch platform {
	case "darwin":
		if mapping.Darwin != "" {
			return mapping.Darwin, nil
		}
	case "linux":
		if mapping.Linux != "" {
			return mapping.Linux, nil
		}
	case "windows":
		if mapping.Windows != "" {
			return mapping.Windows, nil
		}
	}

	// Fallback to logical name if platform-specific mapping not found
	return logicalName, nil
}

// GetPackageMapping retrieves the full mapping for a logical package name.
func (r *DefaultRegistry) GetPackageMapping(ctx context.Context, logicalName string) (*PackageMapping, error) {
	mapping, exists := r.mappings[logicalName]
	if !exists {
		return nil, nil
	}
	return &mapping, nil
}

// ListPackages returns all available package mappings.
func (r *DefaultRegistry) ListPackages(ctx context.Context) ([]PackageMapping, error) {
	var packages []PackageMapping
	for _, mapping := range r.mappings {
		packages = append(packages, mapping)
	}
	return packages, nil
}

// AddPackageMapping adds or updates a package mapping.
func (r *DefaultRegistry) AddPackageMapping(ctx context.Context, mapping PackageMapping) error {
	r.mappings[mapping.LogicalName] = mapping
	return nil
}

// getDefaultMappings returns the default package name mappings.
func getDefaultMappings() map[string]PackageMapping {
	return map[string]PackageMapping{
		"git": {
			LogicalName: "git",
			Darwin:      "git",
			Linux:       "git",
			Windows:     "Git.Git",
		},
		"docker": {
			LogicalName: "docker",
			Darwin:      "colima", // or docker-desktop
			Linux:       "docker.io",
			Windows:     "Docker.DockerDesktop",
		},
		"nodejs": {
			LogicalName: "nodejs",
			Darwin:      "node",
			Linux:       "nodejs",
			Windows:     "OpenJS.NodeJS",
		},
		"python": {
			LogicalName: "python",
			Darwin:      "python@3.12",
			Linux:       "python3",
			Windows:     "Python.Python.3.12",
		},
		"jq": {
			LogicalName: "jq",
			Darwin:      "jq",
			Linux:       "jq",
			Windows:     "jqlang.jq",
		},
	}
}
