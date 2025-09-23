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
	"context"
	"strings"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// GenericServiceDetector provides a basic service detection implementation
// that works on any platform by checking process names
type GenericServiceDetector struct {
	executor executor.Executor
	mapping  *PackageServiceMapping
	health   HealthChecker
}

// NewGenericServiceDetector creates a new generic service detector
func NewGenericServiceDetector(executor executor.Executor, mapping *PackageServiceMapping, health HealthChecker) *GenericServiceDetector {
	return &GenericServiceDetector{
		executor: executor,
		mapping:  mapping,
		health:   health,
	}
}

// IsRunning checks if a service is currently running by process name
func (g *GenericServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	return g.checkProcessName(ctx, serviceName)
}

// GetServiceInfo retrieves basic information about a service
func (g *GenericServiceDetector) GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Name:        serviceName,
		Running:     false,
		Healthy:     false,
		ManagerType: string(ServiceManagerProcess),
		Metadata:    make(map[string]string),
	}

	// Check if process is running
	if running, err := g.checkProcessName(ctx, serviceName); err == nil {
		info.Running = running
	} else {
		return nil, &ServiceDetectionError{
			ServiceName: serviceName,
			Platform:    "generic",
			Cause:       err,
			Suggestion:  "Service may not be installed or accessible. Check if the service process is running.",
		}
	}

	// Add package information if available
	if packageName := g.mapping.GetPackageForService(serviceName); packageName != "" {
		info.Package = &PackageInfo{
			Name:    packageName,
			Manager: "unknown",
		}
	}

	// Perform health check if service is running
	if info.Running {
		if healthConfig := g.mapping.GetDefaultHealthCheck(serviceName); healthConfig != nil {
			if healthResult, err := g.health.CheckCommand(ctx, healthConfig.Command, healthConfig.Timeout); err == nil {
				info.Healthy = healthResult.Healthy
			}
		}
	}

	return info, nil
}

// GetAllServices retrieves information about all detectable services
func (g *GenericServiceDetector) GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error) {
	services := make(map[string]*ServiceInfo)

	// Get all known service names from mapping
	knownServices := g.mapping.GetAllServices()

	for _, serviceName := range knownServices {
		if info, err := g.GetServiceInfo(ctx, serviceName); err == nil {
			services[serviceName] = info
		}
	}

	return services, nil
}

// CheckHealth performs health checks on a service
func (g *GenericServiceDetector) CheckHealth(ctx context.Context, serviceName string, config *HealthCheckConfig) (*HealthResult, error) {
	if config == nil {
		config = g.mapping.GetDefaultHealthCheck(serviceName)
	}

	if config == nil {
		return &HealthResult{
			Healthy: false,
			Error:   "no health check configuration available",
		}, nil
	}

	// Choose appropriate health check method
	if config.Command != "" {
		return g.health.CheckCommand(ctx, config.Command, config.Timeout)
	} else if config.HTTPEndpoint != "" {
		return g.health.CheckHTTP(ctx, config.HTTPEndpoint, config.ExpectedStatus, config.Timeout)
	} else if config.TCPHost != "" && config.TCPPort > 0 {
		return g.health.CheckTCP(ctx, config.TCPHost, config.TCPPort, config.Timeout)
	}

	return &HealthResult{
		Healthy: false,
		Error:   "no valid health check method configured",
	}, nil
}

// checkProcessName checks if a service is running by process name
// This is a basic implementation that should work on most platforms
func (g *GenericServiceDetector) checkProcessName(ctx context.Context, serviceName string) (bool, error) {
	// Try pgrep first (available on most Unix-like systems)
	result, err := g.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err == nil {
		return strings.TrimSpace(result.Stdout) != "", nil
	}

	// Try ps as fallback
	result, err = g.executor.Run(ctx, "ps", []string{"-ef"}, executor.ExecOpts{})
	if err != nil {
		return false, err
	}

	// Simple string matching in process list
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(serviceName)) {
			return true, nil
		}
	}

	return false, nil
}

// Ensure GenericServiceDetector implements ServiceDetector interface
var _ ServiceDetector = (*GenericServiceDetector)(nil)
