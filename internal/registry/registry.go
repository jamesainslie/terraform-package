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

// Package registry provides package name mapping and resolution services.
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
	ResolvePackageName(_ context.Context, logicalName string, platform string) (string, error)

	// GetPackageMapping retrieves the full mapping for a logical package name
	GetPackageMapping(_ context.Context, logicalName string) (*PackageMapping, error)

	// ListPackages returns all available package mappings
	ListPackages(_ context.Context) ([]PackageMapping, error)

	// AddPackageMapping adds or updates a package mapping
	AddPackageMapping(_ context.Context, mapping PackageMapping) error
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
func (r *DefaultRegistry) ResolvePackageName(_ context.Context, logicalName string, platform string) (string, error) {
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
func (r *DefaultRegistry) GetPackageMapping(_ context.Context, logicalName string) (*PackageMapping, error) {
	mapping, exists := r.mappings[logicalName]
	if !exists {
		return nil, nil
	}
	return &mapping, nil
}

// ListPackages returns all available package mappings.
func (r *DefaultRegistry) ListPackages(_ context.Context) ([]PackageMapping, error) {
	packages := make([]PackageMapping, 0, len(r.mappings))
	for _, mapping := range r.mappings {
		packages = append(packages, mapping)
	}
	return packages, nil
}

// AddPackageMapping adds or updates a package mapping.
func (r *DefaultRegistry) AddPackageMapping(_ context.Context, mapping PackageMapping) error {
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
		"wget": {
			LogicalName: "wget",
			Darwin:      "wget",
			Linux:       "wget",
			Windows:     "JernejSimoncic.Wget",
		},
		"hello": {
			LogicalName: "hello",
			Darwin:      "hello",
			Linux:       "hello",
			Windows:     "hello",
		},
	}
}
