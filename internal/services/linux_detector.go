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

//go:build linux

package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// LinuxServiceDetector implements service detection and management for Linux using systemd
type LinuxServiceDetector struct {
	executor executor.Executor
	mapping  *PackageServiceMapping
	health   HealthChecker
}

// NewLinuxServiceDetector creates a new Linux service detector
func NewLinuxServiceDetector(executor executor.Executor, mapping *PackageServiceMapping, health HealthChecker) *LinuxServiceDetector {
	return &LinuxServiceDetector{
		executor: executor,
		mapping:  mapping,
		health:   health,
	}
}

// IsRunning checks if a service is currently running on Linux
func (l *LinuxServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	// Try systemctl first
	result, err := l.executor.Run(ctx, "systemctl", []string{"is-active", serviceName}, executor.ExecOpts{})
	if err == nil && strings.TrimSpace(result.Stdout) == "active" {
		return true, nil
	}

	// Fallback to process name checking
	return l.checkProcessName(ctx, serviceName)
}

// GetServiceInfo retrieves detailed information about a service
func (l *LinuxServiceDetector) GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Name:        serviceName,
		Running:     false,
		Healthy:     false,
		Enabled:     false,
		ManagerType: string(ServiceManagerProcess),
		Metadata:    make(map[string]string),
	}

	// Try systemctl status
	if systemdInfo, err := l.getSystemdServiceInfo(ctx, serviceName); err == nil {
		info.Running = systemdInfo.Active
		info.ManagerType = string(ServiceManagerSystemd)
		info.ProcessID = systemdInfo.PID
		info.Metadata["systemd_status"] = systemdInfo.Status
		info.Metadata["systemd_enabled"] = systemdInfo.Enabled
	} else {
		// Fallback to process checking
		if running, err := l.checkProcessName(ctx, serviceName); err == nil {
			info.Running = running
		} else {
			return nil, &ServiceDetectionError{
				ServiceName: serviceName,
				Platform:    "linux",
				Cause:       err,
				Suggestion:  "Service may not be installed or accessible. Try installing with your package manager.",
			}
		}
	}

	// Add package information if available
	if packageName := l.mapping.GetPackageForService(serviceName); packageName != "" {
		info.Package = &PackageInfo{
			Name:    packageName,
			Manager: "apt", // Default assumption for Linux
		}
	}

	// Check if service is enabled for automatic startup
	if enabled, err := l.IsServiceEnabled(ctx, serviceName); err == nil {
		info.Enabled = enabled
	}

	// Perform health check if service is running
	if info.Running {
		if healthConfig := l.mapping.GetDefaultHealthCheck(serviceName); healthConfig != nil {
			if healthResult, err := l.health.CheckCommand(ctx, healthConfig.Command, healthConfig.Timeout); err == nil {
				info.Healthy = healthResult.Healthy
			}
		}
	}

	return info, nil
}

// GetAllServices retrieves information about all detectable services
func (l *LinuxServiceDetector) GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error) {
	services := make(map[string]*ServiceInfo)

	// Get all known service names from mapping
	knownServices := l.mapping.GetAllServices()

	for _, serviceName := range knownServices {
		if info, err := l.GetServiceInfo(ctx, serviceName); err == nil {
			services[serviceName] = info
		}
	}

	return services, nil
}

// CheckHealth performs health checks on a service
func (l *LinuxServiceDetector) CheckHealth(ctx context.Context, serviceName string, config *HealthCheckConfig) (*HealthResult, error) {
	if config == nil {
		config = l.mapping.GetDefaultHealthCheck(serviceName)
	}

	if config == nil {
		return &HealthResult{
			Healthy: false,
			Error:   "no health check configuration available",
		}, nil
	}

	// Choose appropriate health check method
	if config.Command != "" {
		return l.health.CheckCommand(ctx, config.Command, config.Timeout)
	} else if config.HTTPEndpoint != "" {
		return l.health.CheckHTTP(ctx, config.HTTPEndpoint, config.ExpectedStatus, config.Timeout)
	} else if config.TCPHost != "" && config.TCPPort > 0 {
		return l.health.CheckTCP(ctx, config.TCPHost, config.TCPPort, config.Timeout)
	}

	return &HealthResult{
		Healthy: false,
		Error:   "no valid health check method configured",
	}, nil
}

// SystemdServiceInfo represents systemd service information
type SystemdServiceInfo struct {
	Active  bool
	Status  string
	Enabled string
	PID     string
}

// getSystemdServiceInfo gets detailed info from systemctl
func (l *LinuxServiceDetector) getSystemdServiceInfo(ctx context.Context, serviceName string) (*SystemdServiceInfo, error) {
	// Get service status
	statusResult, err := l.executor.Run(ctx, "systemctl", []string{"is-active", serviceName}, executor.ExecOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %w", err)
	}

	status := strings.TrimSpace(statusResult.Stdout)
	active := status == "active"

	// Get enabled status
	enabledResult, _ := l.executor.Run(ctx, "systemctl", []string{"is-enabled", serviceName}, executor.ExecOpts{})
	enabled := strings.TrimSpace(enabledResult.Stdout)

	info := &SystemdServiceInfo{
		Active:  active,
		Status:  status,
		Enabled: enabled,
	}

	// Try to get PID if service is active
	if active {
		showResult, err := l.executor.Run(ctx, "systemctl", []string{"show", serviceName, "--property=MainPID"}, executor.ExecOpts{})
		if err == nil {
			parts := strings.Split(strings.TrimSpace(showResult.Stdout), "=")
			if len(parts) == 2 {
				info.PID = parts[1]
			}
		}
	}

	return info, nil
}

// checkProcessName checks if a service is running by process name
func (l *LinuxServiceDetector) checkProcessName(ctx context.Context, serviceName string) (bool, error) {
	// Use pgrep to check for running process
	result, err := l.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		// pgrep returns non-zero exit code when no processes found
		return false, nil
	}

	return strings.TrimSpace(result.Stdout) != "", nil
}

// StartService starts a service using systemctl
func (l *LinuxServiceDetector) StartService(ctx context.Context, serviceName string) error {
	result, err := l.executor.Run(ctx, "systemctl", []string{"start", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// StopService stops a service using systemctl
func (l *LinuxServiceDetector) StopService(ctx context.Context, serviceName string) error {
	result, err := l.executor.Run(ctx, "systemctl", []string{"stop", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to stop service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// RestartService restarts a service using systemctl
func (l *LinuxServiceDetector) RestartService(ctx context.Context, serviceName string) error {
	result, err := l.executor.Run(ctx, "systemctl", []string{"restart", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to restart service %s: %s", serviceName, result.Stderr)
	}
	return nil
}


// DisableService disables a service from starting automatically on system startup
func (l *LinuxServiceDetector) DisableService(ctx context.Context, serviceName string) error {
	result, err := l.executor.Run(ctx, "systemctl", []string{"disable", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to disable service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// IsServiceEnabled checks if a service is enabled for automatic startup
func (l *LinuxServiceDetector) IsServiceEnabled(ctx context.Context, serviceName string) (bool, error) {
	result, err := l.executor.Run(ctx, "systemctl", []string{"is-enabled", serviceName}, executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check if service %s is enabled: %w", serviceName, err)
	}

	// systemctl is-enabled returns "enabled" for enabled services
	enabled := strings.TrimSpace(result.Stdout) == "enabled"
	return enabled, nil
}

// SetServiceStartup sets whether a service should start on system startup
func (l *LinuxServiceDetector) SetServiceStartup(ctx context.Context, serviceName string, enabled bool) error {
	if enabled {
		result, err := l.executor.Run(ctx, "systemctl", []string{"enable", serviceName}, executor.ExecOpts{})
		if err != nil {
			return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("failed to enable service %s: %s", serviceName, result.Stderr)
		}
		return nil
	}
	return l.DisableService(ctx, serviceName)
}

// GetServicesForPackage returns service names associated with a package
func (l *LinuxServiceDetector) GetServicesForPackage(packageName string) ([]string, error) {
	return l.mapping.GetServicesForPackage(packageName), nil
}

// GetPackageForService returns the package name associated with a service
func (l *LinuxServiceDetector) GetPackageForService(serviceName string) (string, error) {
	return l.mapping.GetPackageForService(serviceName), nil
}

// Ensure LinuxServiceDetector implements ServiceManager interface
var _ ServiceManager = (*LinuxServiceDetector)(nil)
