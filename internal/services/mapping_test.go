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

package services

import (
	"reflect"
	"testing"
	"time"
)

func TestGetDefaultMapping(t *testing.T) {
	mapping := GetDefaultMapping()

	if mapping == nil {
		t.Fatal("GetDefaultMapping() returned nil")
	}

	if mapping.PackageToServices == nil {
		t.Error("PackageToServices should not be nil")
	}

	if mapping.ServiceToPackage == nil {
		t.Error("ServiceToPackage should not be nil")
	}

	if mapping.DefaultHealthChecks == nil {
		t.Error("DefaultHealthChecks should not be nil")
	}
}

func TestGetServicesForPackage(t *testing.T) {
	mapping := GetDefaultMapping()

	tests := []struct {
		name        string
		packageName string
		expected    []string
	}{
		{
			name:        "colima package",
			packageName: "colima",
			expected:    []string{"colima"},
		},
		{
			name:        "docker package",
			packageName: "docker",
			expected:    []string{"docker", "docker-desktop"},
		},
		{
			name:        "postgresql package",
			packageName: "postgresql",
			expected:    []string{"postgres", "postgresql"},
		},
		{
			name:        "nonexistent package",
			packageName: "nonexistent",
			expected:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.GetServicesForPackage(tt.packageName)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetServicesForPackage(%q) = %v, want %v", tt.packageName, result, tt.expected)
			}
		})
	}
}

func TestGetPackageForService(t *testing.T) {
	mapping := GetDefaultMapping()

	tests := []struct {
		name        string
		serviceName string
		expected    string
	}{
		{
			name:        "colima service",
			serviceName: "colima",
			expected:    "colima",
		},
		{
			name:        "docker service",
			serviceName: "docker",
			expected:    "docker",
		},
		{
			name:        "docker-desktop service",
			serviceName: "docker-desktop",
			expected:    "docker",
		},
		{
			name:        "postgres service",
			serviceName: "postgres",
			expected:    "postgresql",
		},
		{
			name:        "nonexistent service",
			serviceName: "nonexistent",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.GetPackageForService(tt.serviceName)
			if result != tt.expected {
				t.Errorf("GetPackageForService(%q) = %v, want %v", tt.serviceName, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultHealthCheck(t *testing.T) {
	mapping := GetDefaultMapping()

	tests := []struct {
		name        string
		serviceName string
		expectNil   bool
		hasCommand  bool
		hasHTTP     bool
		hasTimeout  bool
	}{
		{
			name:        "colima health check",
			serviceName: "colima",
			expectNil:   false,
			hasCommand:  true,
			hasHTTP:     false,
			hasTimeout:  true,
		},
		{
			name:        "docker health check",
			serviceName: "docker",
			expectNil:   false,
			hasCommand:  false,
			hasHTTP:     true,
			hasTimeout:  true,
		},
		{
			name:        "nonexistent service health check",
			serviceName: "nonexistent",
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.GetDefaultHealthCheck(tt.serviceName)

			if tt.expectNil {
				if result != nil {
					t.Errorf("GetDefaultHealthCheck(%q) = %v, want nil", tt.serviceName, result)
				}
				return
			}

			if result == nil {
				t.Errorf("GetDefaultHealthCheck(%q) = nil, want non-nil", tt.serviceName)
				return
			}

			if tt.hasCommand && result.Command == "" {
				t.Errorf("Expected command to be set for service %q", tt.serviceName)
			}

			if tt.hasHTTP && result.HTTPEndpoint == "" {
				t.Errorf("Expected HTTP endpoint to be set for service %q", tt.serviceName)
			}

			if tt.hasTimeout && result.Timeout == 0 {
				t.Errorf("Expected timeout to be set for service %q", tt.serviceName)
			}
		})
	}
}

func TestAddAndRemoveMapping(t *testing.T) {
	mapping := GetDefaultMapping()

	// Test adding a new mapping
	testPackage := "test-package"
	testServices := []string{"test-service1", "test-service2"}

	mapping.AddMapping(testPackage, testServices)

	// Verify the package-to-service mapping
	result := mapping.GetServicesForPackage(testPackage)
	if !reflect.DeepEqual(result, testServices) {
		t.Errorf("After adding, GetServicesForPackage(%q) = %v, want %v", testPackage, result, testServices)
	}

	// Verify the service-to-package mappings
	for _, service := range testServices {
		pkg := mapping.GetPackageForService(service)
		if pkg != testPackage {
			t.Errorf("After adding, GetPackageForService(%q) = %q, want %q", service, pkg, testPackage)
		}
	}

	// Test removing the mapping
	mapping.RemoveMapping(testPackage)

	// Verify the package-to-service mapping is removed
	result = mapping.GetServicesForPackage(testPackage)
	if result != nil {
		t.Errorf("After removing, GetServicesForPackage(%q) = %v, want nil", testPackage, result)
	}

	// Verify the service-to-package mappings are removed
	for _, service := range testServices {
		pkg := mapping.GetPackageForService(service)
		if pkg != "" {
			t.Errorf("After removing, GetPackageForService(%q) = %q, want empty string", service, pkg)
		}
	}
}

func TestFindServiceByName(t *testing.T) {
	mapping := GetDefaultMapping()

	tests := []struct {
		name     string
		query    string
		expected []string
		minCount int // minimum expected matches
	}{
		{
			name:     "exact match - colima",
			query:    "colima",
			expected: []string{"colima"},
		},
		{
			name:     "exact match - docker",
			query:    "docker",
			expected: []string{"docker"},
		},
		{
			name:     "partial match - post",
			query:    "post",
			minCount: 1, // should match postgres, postgresql
		},
		{
			name:     "case insensitive - DOCKER",
			query:    "DOCKER",
			minCount: 1, // should match docker, docker-desktop
		},
		{
			name:     "no match",
			query:    "nonexistentservice",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapping.FindServiceByName(tt.query)

			if tt.expected != nil {
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("FindServiceByName(%q) = %v, want %v", tt.query, result, tt.expected)
				}
			} else if len(result) < tt.minCount {
				t.Errorf("FindServiceByName(%q) returned %d matches, want at least %d", tt.query, len(result), tt.minCount)
			}
		})
	}
}

func TestGetAllServices(t *testing.T) {
	mapping := GetDefaultMapping()

	services := mapping.GetAllServices()

	if len(services) == 0 {
		t.Error("GetAllServices() returned empty slice, expected some services")
	}

	// Check for some expected services
	expectedServices := []string{"colima", "docker", "postgres", "redis"}
	serviceMap := make(map[string]bool)
	for _, service := range services {
		serviceMap[service] = true
	}

	for _, expected := range expectedServices {
		if !serviceMap[expected] {
			t.Errorf("Expected service %q to be in GetAllServices() result", expected)
		}
	}
}

func TestGetAllPackages(t *testing.T) {
	mapping := GetDefaultMapping()

	packages := mapping.GetAllPackages()

	if len(packages) == 0 {
		t.Error("GetAllPackages() returned empty slice, expected some packages")
	}

	// Check for some expected packages
	expectedPackages := []string{"colima", "docker", "postgresql", "redis"}
	packageMap := make(map[string]bool)
	for _, pkg := range packages {
		packageMap[pkg] = true
	}

	for _, expected := range expectedPackages {
		if !packageMap[expected] {
			t.Errorf("Expected package %q to be in GetAllPackages() result", expected)
		}
	}
}

func TestHealthCheckTimeout(t *testing.T) {
	mapping := GetDefaultMapping()

	// Test that health check timeouts are reasonable
	for serviceName := range mapping.DefaultHealthChecks {
		healthCheck := mapping.GetDefaultHealthCheck(serviceName)
		if healthCheck == nil {
			continue
		}

		if healthCheck.Timeout <= 0 {
			t.Errorf("Service %q has invalid timeout: %v", serviceName, healthCheck.Timeout)
		}

		if healthCheck.Timeout > 30*time.Second {
			t.Errorf("Service %q has unreasonably long timeout: %v", serviceName, healthCheck.Timeout)
		}
	}
}
