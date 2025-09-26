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
	"strings"
)

// PackageServiceMapping defines the mapping between packages and their services
type PackageServiceMapping struct {
	// Package name to service names mapping
	PackageToServices map[string][]string

	// Service name to package name mapping
	ServiceToPackage map[string]string

	// Default health check configurations for services
	DefaultHealthChecks map[string]*HealthCheckConfig
}

// GetDefaultMapping returns the default package-to-service mapping
func GetDefaultMapping() *PackageServiceMapping {
	packageToServices := map[string][]string{
		// Container runtimes
		"colima": {"colima"},
		"docker": {"docker", "docker-desktop"},
		"podman": {"podman"},
		"lima":   {"lima"},

		// Databases
		"postgresql":    {"postgres", "postgresql"},
		"mysql":         {"mysqld", "mysql"},
		"redis":         {"redis-server", "redis"},
		"mongodb":       {"mongod", "mongodb"},
		"sqlite":        {"sqlite3"},
		"cassandra":     {"cassandra"},
		"elasticsearch": {"elasticsearch"},

		// Web servers
		"nginx":   {"nginx"},
		"apache2": {"apache2", "httpd"},
		"caddy":   {"caddy"},
		"traefik": {"traefik"},

		// Message queues
		"rabbitmq": {"rabbitmq-server"},
		"kafka":    {"kafka"},
		"nats":     {"nats-server"},

		// Monitoring & observability
		"prometheus": {"prometheus"},
		"grafana":    {"grafana-server"},
		"jaeger":     {"jaeger"},
		"zipkin":     {"zipkin"},

		// Development tools
		"node":   {"node"},
		"python": {"python", "python3"},
		"java":   {"java"},
		"golang": {"go"},

		// Version control
		"git":       {"git"},
		"mercurial": {"hg"},

		// Search engines
		"solr":       {"solr"},
		"opensearch": {"opensearch"},

		// Cache systems
		"memcached": {"memcached"},
		"hazelcast": {"hazelcast"},

		// API gateways
		"kong":  {"kong"},
		"envoy": {"envoy"},

		// Service mesh
		"istio":  {"istio-proxy", "pilot-discovery"},
		"consul": {"consul"},
		"vault":  {"vault"},

		// Build tools
		"jenkins":       {"jenkins"},
		"gitlab-runner": {"gitlab-runner"},

		// File systems
		"minio": {"minio"},
		"samba": {"smbd", "nmbd"},
	}

	// Create reverse mapping
	serviceToPackage := make(map[string]string)
	for pkg, services := range packageToServices {
		for _, service := range services {
			serviceToPackage[service] = pkg
		}
	}

	// Default health check configurations
	defaultHealthChecks := map[string]*HealthCheckConfig{
		"colima": {
			Command: "colima status",
			Timeout: 10000000000, // 10 seconds in nanoseconds
		},
		"docker": {
			HTTPEndpoint:   "http://localhost:2375/_ping",
			ExpectedStatus: 200,
			Timeout:        5000000000, // 5 seconds
		},
		"postgres": {
			Command: "pg_isready -h localhost -p 5432",
			Timeout: 5000000000,
		},
		"postgresql": {
			Command: "pg_isready -h localhost -p 5432",
			Timeout: 5000000000,
		},
		"mysql": {
			Command: "mysqladmin ping -h localhost",
			Timeout: 5000000000,
		},
		"mysqld": {
			Command: "mysqladmin ping -h localhost",
			Timeout: 5000000000,
		},
		"redis": {
			Command: "redis-cli ping",
			Timeout: 3000000000, // 3 seconds
		},
		"redis-server": {
			Command: "redis-cli ping",
			Timeout: 3000000000,
		},
		"nginx": {
			HTTPEndpoint:   "http://localhost:80",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
		"elasticsearch": {
			HTTPEndpoint:   "http://localhost:9200/_health",
			ExpectedStatus: 200,
			Timeout:        10000000000, // ES can be slow to start
		},
		"prometheus": {
			HTTPEndpoint:   "http://localhost:9090/-/healthy",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
		"grafana-server": {
			HTTPEndpoint:   "http://localhost:3000/api/health",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
		"mongodb": {
			Command: "mongosh --eval \"db.adminCommand('ping')\"",
			Timeout: 5000000000,
		},
		"mongod": {
			Command: "mongosh --eval \"db.adminCommand('ping')\"",
			Timeout: 5000000000,
		},
		"rabbitmq-server": {
			HTTPEndpoint:   "http://localhost:15672/api/overview",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
		"consul": {
			HTTPEndpoint:   "http://localhost:8500/v1/status/leader",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
		"vault": {
			HTTPEndpoint:   "http://localhost:8200/v1/sys/health",
			ExpectedStatus: 200,
			Timeout:        5000000000,
		},
	}

	return &PackageServiceMapping{
		PackageToServices:   packageToServices,
		ServiceToPackage:    serviceToPackage,
		DefaultHealthChecks: defaultHealthChecks,
	}
}

// GetServicesForPackage returns the service names associated with a package
func (m *PackageServiceMapping) GetServicesForPackage(packageName string) []string {
	services, exists := m.PackageToServices[packageName]
	if !exists {
		return nil
	}
	// Return a copy to prevent modification
	result := make([]string, len(services))
	copy(result, services)
	return result
}

// GetPackageForService returns the package name that provides a service
func (m *PackageServiceMapping) GetPackageForService(serviceName string) string {
	return m.ServiceToPackage[serviceName]
}

// GetDefaultHealthCheck returns the default health check configuration for a service
func (m *PackageServiceMapping) GetDefaultHealthCheck(serviceName string) *HealthCheckConfig {
	config, exists := m.DefaultHealthChecks[serviceName]
	if !exists {
		return nil
	}
	// Return a copy to prevent modification
	return &HealthCheckConfig{
		Command:        config.Command,
		HTTPEndpoint:   config.HTTPEndpoint,
		ExpectedStatus: config.ExpectedStatus,
		TCPHost:        config.TCPHost,
		TCPPort:        config.TCPPort,
		Timeout:        config.Timeout,
		RetryCount:     config.RetryCount,
		RetryInterval:  config.RetryInterval,
	}
}

// AddMapping adds a new package-to-service mapping
func (m *PackageServiceMapping) AddMapping(packageName string, serviceNames []string) {
	m.PackageToServices[packageName] = serviceNames
	for _, serviceName := range serviceNames {
		m.ServiceToPackage[serviceName] = packageName
	}
}

// RemoveMapping removes a package-to-service mapping
func (m *PackageServiceMapping) RemoveMapping(packageName string) {
	services := m.PackageToServices[packageName]
	for _, serviceName := range services {
		delete(m.ServiceToPackage, serviceName)
	}
	delete(m.PackageToServices, packageName)
}

// FindServiceByName attempts to find a service name using fuzzy matching
func (m *PackageServiceMapping) FindServiceByName(query string) []string {
	var matches []string
	query = strings.ToLower(query)

	// First, try exact matches
	for serviceName := range m.ServiceToPackage {
		if strings.ToLower(serviceName) == query {
			matches = append(matches, serviceName)
		}
	}

	// If no exact matches, try partial matches
	if len(matches) == 0 {
		for serviceName := range m.ServiceToPackage {
			if strings.Contains(strings.ToLower(serviceName), query) {
				matches = append(matches, serviceName)
			}
		}
	}

	return matches
}

// GetAllServices returns all known service names
func (m *PackageServiceMapping) GetAllServices() []string {
	services := make([]string, 0, len(m.ServiceToPackage))
	for serviceName := range m.ServiceToPackage {
		services = append(services, serviceName)
	}
	return services
}

// GetAllPackages returns all known package names
func (m *PackageServiceMapping) GetAllPackages() []string {
	packages := make([]string, 0, len(m.PackageToServices))
	for packageName := range m.PackageToServices {
		packages = append(packages, packageName)
	}
	return packages
}
