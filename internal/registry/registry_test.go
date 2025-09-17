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

package registry

import (
	"context"
	"testing"
)

func TestDefaultRegistry_ResolvePackageName(t *testing.T) {
	registry := NewDefaultRegistry()
	ctx := context.Background()

	tests := []struct {
		logicalName string
		platform    string
		expected    string
	}{
		{"git", "darwin", "git"},
		{"git", "linux", "git"},
		{"git", "windows", "Git.Git"},
		{"docker", "darwin", "colima"},
		{"docker", "linux", "docker.io"},
		{"docker", "windows", "Docker.DockerDesktop"},
		{"nonexistent", "darwin", "nonexistent"}, // Should return logical name as fallback
	}

	for _, test := range tests {
		result, err := registry.ResolvePackageName(ctx, test.logicalName, test.platform)
		if err != nil {
			t.Errorf("Unexpected error resolving %s for %s: %v", test.logicalName, test.platform, err)
			continue
		}

		if result != test.expected {
			t.Errorf("Expected %s for %s on %s, got %s", test.expected, test.logicalName, test.platform, result)
		}
	}
}

func TestDefaultRegistry_GetPackageMapping(t *testing.T) {
	registry := NewDefaultRegistry()
	ctx := context.Background()

	// Test existing package
	mapping, err := registry.GetPackageMapping(ctx, "git")
	if err != nil {
		t.Fatalf("Unexpected error getting git mapping: %v", err)
	}

	if mapping == nil {
		t.Fatal("Expected mapping for git, got nil")
	}

	if mapping.LogicalName != "git" {
		t.Errorf("Expected logical name 'git', got '%s'", mapping.LogicalName)
	}

	// Test non-existent package
	mapping, err = registry.GetPackageMapping(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Unexpected error getting nonexistent mapping: %v", err)
	}

	if mapping != nil {
		t.Error("Expected nil mapping for nonexistent package")
	}
}

func TestDefaultRegistry_ListPackages(t *testing.T) {
	registry := NewDefaultRegistry()
	ctx := context.Background()

	packages, err := registry.ListPackages(ctx)
	if err != nil {
		t.Fatalf("Unexpected error listing packages: %v", err)
	}

	if len(packages) == 0 {
		t.Error("Expected at least one package mapping")
	}

	// Check that git is in the list
	found := false
	for _, pkg := range packages {
		if pkg.LogicalName == "git" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find git package in the list")
	}
}

func TestDefaultRegistry_AddPackageMapping(t *testing.T) {
	registry := NewDefaultRegistry()
	ctx := context.Background()

	newMapping := PackageMapping{
		LogicalName: "testpkg",
		Darwin:      "testpkg-mac",
		Linux:       "testpkg-linux",
		Windows:     "TestPkg.TestPkg",
	}

	err := registry.AddPackageMapping(ctx, newMapping)
	if err != nil {
		t.Fatalf("Unexpected error adding package mapping: %v", err)
	}

	// Verify the mapping was added
	retrieved, err := registry.GetPackageMapping(ctx, "testpkg")
	if err != nil {
		t.Fatalf("Unexpected error retrieving added mapping: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve added mapping, got nil")
	}

	if retrieved.Darwin != "testpkg-mac" {
		t.Errorf("Expected Darwin name 'testpkg-mac', got '%s'", retrieved.Darwin)
	}
}
