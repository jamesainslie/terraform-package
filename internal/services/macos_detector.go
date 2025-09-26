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

//go:build darwin

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// MacOSServiceDetector implements service detection and management for macOS using launchd and brew services
type MacOSServiceDetector struct {
	executor executor.Executor
	mapping  *PackageServiceMapping
	health   HealthChecker
}

// NewMacOSServiceDetector creates a new macOS service detector
func NewMacOSServiceDetector(executor executor.Executor, mapping *PackageServiceMapping, health HealthChecker) *MacOSServiceDetector {
	return &MacOSServiceDetector{
		executor: executor,
		mapping:  mapping,
		health:   health,
	}
}

// BrewService represents a service from brew services output
type BrewService struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	User     string `json:"user"`
	PID      string `json:"pid"`
	ExitCode *int   `json:"exit_code"`
}

// LaunchdService represents a service from launchctl output
type LaunchdService struct {
	PID    string
	Status string
	Label  string
}

// IsRunning checks if a service is currently running on macOS
func (m *MacOSServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	// Try multiple detection methods in order of preference

	// 1. Check brew services first (most reliable for brew-installed services)
	if running, err := m.checkBrewServices(ctx, serviceName); err == nil {
		return running, nil
	}

	// 2. Check launchctl
	if running, err := m.checkLaunchctl(ctx, serviceName); err == nil {
		return running, nil
	}

	// 3. Check by process name as fallback
	return m.checkProcessName(ctx, serviceName)
}

// GetServiceInfo retrieves detailed information about a service
func (m *MacOSServiceDetector) GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Name:        serviceName,
		Running:     false,
		Healthy:     false,
		Enabled:     false,
		ManagerType: string(ServiceManagerProcess), // default fallback
		Metadata:    make(map[string]string),
	}

	// Try to get info from brew services first
	if brewInfo, err := m.getBrewServiceInfo(ctx, serviceName); err == nil {
		info.Running = brewInfo.Status == "started"
		info.ManagerType = string(ServiceManagerBrewServices)
		info.ProcessID = brewInfo.PID
		info.Metadata["brew_status"] = brewInfo.Status
		info.Metadata["brew_user"] = brewInfo.User
		if brewInfo.ExitCode != nil {
			info.Metadata["exit_code"] = strconv.Itoa(*brewInfo.ExitCode)
		}
	} else {
		// Fallback to launchctl
		if launchdInfo, err := m.getLaunchdServiceInfo(ctx, serviceName); err == nil {
			info.Running = launchdInfo.Status != "-"
			info.ManagerType = string(ServiceManagerLaunchd)
			info.ProcessID = launchdInfo.PID
			info.Metadata["launchd_status"] = launchdInfo.Status
			info.Metadata["launchd_label"] = launchdInfo.Label
		} else {
			// Final fallback to process checking
			if running, err := m.checkProcessName(ctx, serviceName); err == nil {
				info.Running = running
				if running {
					if pid, err := m.getProcessPID(ctx, serviceName); err == nil {
						info.ProcessID = pid
					}
				}
			} else {
				return nil, &ServiceDetectionError{
					ServiceName: serviceName,
					Platform:    "darwin",
					Cause:       err,
					Suggestion:  "Service may not be installed or accessible. Try installing with brew or check service name.",
				}
			}
		}
	}

	// Add package information if available
	if packageName := m.mapping.GetPackageForService(serviceName); packageName != "" {
		info.Package = &PackageInfo{
			Name:    packageName,
			Manager: "brew", // Assume brew on macOS
		}

		// Try to get package version
		if version, err := m.getPackageVersion(ctx, packageName); err == nil {
			info.Package.Version = version
			info.Version = version
		}
	}

	// Check if service is enabled for automatic startup
	if enabled, err := m.IsServiceEnabled(ctx, serviceName); err == nil {
		info.Enabled = enabled
	}

	// Perform health check if service is running
	if info.Running {
		if healthConfig := m.mapping.GetDefaultHealthCheck(serviceName); healthConfig != nil {
			if healthResult, err := m.health.CheckCommand(ctx, healthConfig.Command, healthConfig.Timeout); err == nil {
				info.Healthy = healthResult.Healthy
			}
		}
	}

	return info, nil
}

// GetAllServices retrieves information about all detectable services
func (m *MacOSServiceDetector) GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error) {
	services := make(map[string]*ServiceInfo)

	// Get all known service names from mapping
	knownServices := m.mapping.GetAllServices()

	for _, serviceName := range knownServices {
		if info, err := m.GetServiceInfo(ctx, serviceName); err == nil {
			services[serviceName] = info
		}
	}

	return services, nil
}

// CheckHealth performs health checks on a service
func (m *MacOSServiceDetector) CheckHealth(ctx context.Context, serviceName string, config *HealthCheckConfig) (*HealthResult, error) {
	if config == nil {
		// Use default health check if available
		config = m.mapping.GetDefaultHealthCheck(serviceName)
	}

	if config == nil {
		return &HealthResult{
			Healthy: false,
			Error:   "no health check configuration available",
		}, nil
	}

	// Choose appropriate health check method
	if config.Command != "" {
		return m.health.CheckCommand(ctx, config.Command, config.Timeout)
	} else if config.HTTPEndpoint != "" {
		return m.health.CheckHTTP(ctx, config.HTTPEndpoint, config.ExpectedStatus, config.Timeout)
	} else if config.TCPHost != "" && config.TCPPort > 0 {
		return m.health.CheckTCP(ctx, config.TCPHost, config.TCPPort, config.Timeout)
	}

	return &HealthResult{
		Healthy: false,
		Error:   "no valid health check method configured",
	}, nil
}

// checkBrewServices checks if a service is running via brew services
func (m *MacOSServiceDetector) checkBrewServices(ctx context.Context, serviceName string) (bool, error) {
	brewInfo, err := m.getBrewServiceInfo(ctx, serviceName)
	if err != nil {
		return false, err
	}
	return brewInfo.Status == "started", nil
}

// getBrewServiceInfo gets detailed info from brew services
func (m *MacOSServiceDetector) getBrewServiceInfo(ctx context.Context, serviceName string) (*BrewService, error) {
	result, err := m.executor.Run(ctx, "brew", []string{"services", "list", "--json"}, executor.ExecOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to execute brew services list: %w", err)
	}
	output := result.Stdout

	var services []BrewService
	if err := json.Unmarshal([]byte(output), &services); err != nil {
		return nil, fmt.Errorf("failed to parse brew services output: %w", err)
	}

	for _, service := range services {
		if service.Name == serviceName {
			return &service, nil
		}
	}

	return nil, fmt.Errorf("service %s not found in brew services", serviceName)
}

// checkLaunchctl checks if a service is running via launchctl
func (m *MacOSServiceDetector) checkLaunchctl(ctx context.Context, serviceName string) (bool, error) {
	launchdInfo, err := m.getLaunchdServiceInfo(ctx, serviceName)
	if err != nil {
		return false, err
	}
	return launchdInfo.Status != "-", nil
}

// getLaunchdServiceInfo gets detailed info from launchctl
func (m *MacOSServiceDetector) getLaunchdServiceInfo(ctx context.Context, serviceName string) (*LaunchdService, error) {
	// Try different launchctl label patterns
	labelPatterns := []string{
		serviceName,
		"homebrew.mxcl." + serviceName,
		"org.postgresql.postgres", // Special case for postgres
		"com.docker.docker",       // Special case for docker
	}

	for _, label := range labelPatterns {
		result, err := m.executor.Run(ctx, "launchctl", []string{"list", label}, executor.ExecOpts{})
		if err != nil {
			continue // Try next pattern
		}
		output := result.Stdout

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) >= 1 {
			fields := strings.Fields(lines[0])
			if len(fields) >= 3 {
				return &LaunchdService{
					PID:    fields[0],
					Status: fields[1],
					Label:  fields[2],
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("service %s not found in launchctl", serviceName)
}

// checkProcessName checks if a service is running by process name
func (m *MacOSServiceDetector) checkProcessName(ctx context.Context, serviceName string) (bool, error) {
	// Use pgrep to check for running process
	result, err := m.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		// pgrep returns non-zero exit code when no processes found
		return false, nil
	}

	return strings.TrimSpace(result.Stdout) != "", nil
}

// getProcessPID gets the PID of a process by name
func (m *MacOSServiceDetector) getProcessPID(ctx context.Context, serviceName string) (string, error) {
	result, err := m.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		return "", err
	}

	pids := strings.Fields(strings.TrimSpace(result.Stdout))
	if len(pids) > 0 {
		return pids[0], nil // Return first PID
	}

	return "", fmt.Errorf("no PID found for service %s", serviceName)
}

// getPackageVersion gets the version of an installed package
func (m *MacOSServiceDetector) getPackageVersion(ctx context.Context, packageName string) (string, error) {
	// Try brew list with version
	result, err := m.executor.Run(ctx, "brew", []string{"list", "--versions", packageName}, executor.ExecOpts{})
	if err != nil {
		return "", err
	}

	// Parse output like "package 1.2.3"
	fields := strings.Fields(strings.TrimSpace(result.Stdout))
	if len(fields) >= 2 {
		return fields[1], nil
	}

	return "", fmt.Errorf("could not parse version for package %s", packageName)
}

// StartService starts a service using the appropriate service manager
func (m *MacOSServiceDetector) StartService(ctx context.Context, serviceName string) error {
	// Try brew services first
	result, err := m.executor.Run(ctx, "brew", []string{"services", "start", serviceName}, executor.ExecOpts{})
	if err == nil && result.ExitCode == 0 {
		return nil
	}

	// Fallback to launchctl
	result, err = m.executor.Run(ctx, "launchctl", []string{"start", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// StopService stops a service using the appropriate service manager
func (m *MacOSServiceDetector) StopService(ctx context.Context, serviceName string) error {
	// Try brew services first
	result, err := m.executor.Run(ctx, "brew", []string{"services", "stop", serviceName}, executor.ExecOpts{})
	if err == nil && result.ExitCode == 0 {
		return nil
	}

	// Fallback to launchctl
	result, err = m.executor.Run(ctx, "launchctl", []string{"stop", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to stop service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// RestartService restarts a service using the appropriate service manager
func (m *MacOSServiceDetector) RestartService(ctx context.Context, serviceName string) error {
	// Try brew services first
	result, err := m.executor.Run(ctx, "brew", []string{"services", "restart", serviceName}, executor.ExecOpts{})
	if err == nil && result.ExitCode == 0 {
		return nil
	}

	// Fallback to stop then start
	if err := m.StopService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to stop service %s during restart: %w", serviceName, err)
	}
	if err := m.StartService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to start service %s during restart: %w", serviceName, err)
	}
	return nil
}

// DisableService disables a service from starting automatically on system startup
func (m *MacOSServiceDetector) DisableService(ctx context.Context, serviceName string) error {
	// Try brew services first
	result, err := m.executor.Run(ctx, "brew", []string{"services", "stop", serviceName}, executor.ExecOpts{})
	if err == nil && result.ExitCode == 0 {
		return nil
	}

	// For launchd services, we need to unload the plist
	result, err = m.executor.Run(ctx, "launchctl", []string{"unload", "-w", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)}, executor.ExecOpts{})
	if err != nil {
		// Try user domain
		result, err = m.executor.Run(ctx, "launchctl", []string{"unload", "-w", fmt.Sprintf("~/Library/LaunchAgents/%s.plist", serviceName)}, executor.ExecOpts{})
		if err != nil {
			return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
		}
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to disable service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// IsServiceEnabled checks if a service is enabled for automatic startup
func (m *MacOSServiceDetector) IsServiceEnabled(ctx context.Context, serviceName string) (bool, error) {
	// Try brew services first
	result, err := m.executor.Run(ctx, "brew", []string{"services", "list", "--json"}, executor.ExecOpts{})
	if err == nil && result.ExitCode == 0 {
		var services []BrewService
		if err := json.Unmarshal([]byte(result.Stdout), &services); err == nil {
			for _, service := range services {
				if service.Name == serviceName {
					return service.Status == "started", nil
				}
			}
		}
	}

	// For launchd services, check if plist is loaded
	result, err = m.executor.Run(ctx, "launchctl", []string{"list"}, executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check if service %s is enabled: %w", serviceName, err)
	}

	// Check if service appears in launchctl list
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, serviceName) {
			return true, nil
		}
	}
	return false, nil
}

// SetServiceStartup sets whether a service should start on system startup
func (m *MacOSServiceDetector) SetServiceStartup(ctx context.Context, serviceName string, enabled bool) error {
	if enabled {
		// Try brew services first
		result, err := m.executor.Run(ctx, "brew", []string{"services", "start", serviceName}, executor.ExecOpts{})
		if err == nil && result.ExitCode == 0 {
			return nil
		}

		// For launchd services, we need to load the plist
		// This is a simplified approach - in practice, you'd need to handle plist paths
		result, err = m.executor.Run(ctx, "launchctl", []string{"load", "-w", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)}, executor.ExecOpts{})
		if err != nil {
			// Try user domain
			result, err = m.executor.Run(ctx, "launchctl", []string{"load", "-w", fmt.Sprintf("~/Library/LaunchAgents/%s.plist", serviceName)}, executor.ExecOpts{})
			if err != nil {
				return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
			}
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("failed to enable service %s: %s", serviceName, result.Stderr)
		}
		return nil
	}
	return m.DisableService(ctx, serviceName)
}

// GetServicesForPackage returns service names associated with a package
func (m *MacOSServiceDetector) GetServicesForPackage(packageName string) ([]string, error) {
	return m.mapping.GetServicesForPackage(packageName), nil
}

// GetPackageForService returns the package name associated with a service
func (m *MacOSServiceDetector) GetPackageForService(serviceName string) (string, error) {
	return m.mapping.GetPackageForService(serviceName), nil
}

// Ensure MacOSServiceDetector implements ServiceManager interface
var _ ServiceManager = (*MacOSServiceDetector)(nil)
