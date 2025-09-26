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

//go:build windows

package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// WindowsServiceDetector implements service detection and management for Windows using Windows Services
type WindowsServiceDetector struct {
	executor executor.Executor
	mapping  *PackageServiceMapping
	health   HealthChecker
}

// NewWindowsServiceDetector creates a new Windows service detector
func NewWindowsServiceDetector(executor executor.Executor, mapping *PackageServiceMapping, health HealthChecker) *WindowsServiceDetector {
	return &WindowsServiceDetector{
		executor: executor,
		mapping:  mapping,
		health:   health,
	}
}

// IsRunning checks if a service is currently running on Windows
func (w *WindowsServiceDetector) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	// Use PowerShell Get-Service to check service status
	command := "powershell"
	args := []string{
		"-Command",
		"Get-Service -Name " + serviceName + " -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Status",
	}

	result, err := w.executor.Run(ctx, command, args, executor.ExecOpts{})
	if err != nil {
		// Fallback to process name checking
		return w.checkProcessName(ctx, serviceName)
	}

	status := strings.TrimSpace(result.Stdout)
	return status == "Running", nil
}

// GetServiceInfo retrieves detailed information about a service
func (w *WindowsServiceDetector) GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error) {
	info := &ServiceInfo{
		Name:        serviceName,
		Running:     false,
		Healthy:     false,
		Enabled:     false,
		ManagerType: string(ServiceManagerWindowsServices),
		Metadata:    make(map[string]string),
	}

	// Try to get Windows service info
	if winInfo, err := w.getWindowsServiceInfo(ctx, serviceName); err == nil {
		info.Running = winInfo.Status == "Running"
		info.Metadata["windows_status"] = winInfo.Status
		info.Metadata["windows_start_type"] = winInfo.StartType
	} else {
		// Fallback to process checking
		if running, err := w.checkProcessName(ctx, serviceName); err == nil {
			info.Running = running
			info.ManagerType = string(ServiceManagerProcess)
		} else {
			return nil, &ServiceDetectionError{
				ServiceName: serviceName,
				Platform:    "windows",
				Cause:       err,
				Suggestion:  "Service may not be installed or accessible. Try installing with winget or chocolatey.",
			}
		}
	}

	// Add package information if available
	if packageName := w.mapping.GetPackageForService(serviceName); packageName != "" {
		info.Package = &PackageInfo{
			Name:    packageName,
			Manager: "winget", // Default assumption for Windows
		}
	}

	// Check if service is enabled for automatic startup
	if enabled, err := w.IsServiceEnabled(ctx, serviceName); err == nil {
		info.Enabled = enabled
	}

	// Perform health check if service is running
	if info.Running {
		if healthConfig := w.mapping.GetDefaultHealthCheck(serviceName); healthConfig != nil {
			if healthResult, err := w.health.CheckCommand(ctx, healthConfig.Command, healthConfig.Timeout); err == nil {
				info.Healthy = healthResult.Healthy
			}
		}
	}

	return info, nil
}

// GetAllServices retrieves information about all detectable services
func (w *WindowsServiceDetector) GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error) {
	services := make(map[string]*ServiceInfo)

	// Get all known service names from mapping
	knownServices := w.mapping.GetAllServices()

	for _, serviceName := range knownServices {
		if info, err := w.GetServiceInfo(ctx, serviceName); err == nil {
			services[serviceName] = info
		}
	}

	return services, nil
}

// CheckHealth performs health checks on a service
func (w *WindowsServiceDetector) CheckHealth(ctx context.Context, serviceName string, config *HealthCheckConfig) (*HealthResult, error) {
	if config == nil {
		config = w.mapping.GetDefaultHealthCheck(serviceName)
	}

	if config == nil {
		return &HealthResult{
			Healthy: false,
			Error:   "no health check configuration available",
		}, nil
	}

	// Choose appropriate health check method
	if config.Command != "" {
		return w.health.CheckCommand(ctx, config.Command, config.Timeout)
	} else if config.HTTPEndpoint != "" {
		return w.health.CheckHTTP(ctx, config.HTTPEndpoint, config.ExpectedStatus, config.Timeout)
	} else if config.TCPHost != "" && config.TCPPort > 0 {
		return w.health.CheckTCP(ctx, config.TCPHost, config.TCPPort, config.Timeout)
	}

	return &HealthResult{
		Healthy: false,
		Error:   "no valid health check method configured",
	}, nil
}

// WindowsServiceInfo represents Windows service information
type WindowsServiceInfo struct {
	Status    string
	StartType string
}

// getWindowsServiceInfo gets detailed info from Windows Services
func (w *WindowsServiceDetector) getWindowsServiceInfo(ctx context.Context, serviceName string) (*WindowsServiceInfo, error) {
	command := "powershell"
	args := []string{
		"-Command",
		"Get-Service -Name " + serviceName + " -ErrorAction SilentlyContinue | Select-Object Status, StartType | ConvertTo-Json",
	}

	result, err := w.executor.Run(ctx, command, args, executor.ExecOpts{})
	if err != nil {
		return nil, err
	}

	// For now, return a basic implementation
	// In a full implementation, you would parse the JSON output
	info := &WindowsServiceInfo{
		Status:    strings.TrimSpace(result.Stdout),
		StartType: "Unknown",
	}

	return info, nil
}

// checkProcessName checks if a service is running by process name
func (w *WindowsServiceDetector) checkProcessName(ctx context.Context, serviceName string) (bool, error) {
	command := "powershell"
	args := []string{
		"-Command",
		"Get-Process -Name " + serviceName + " -ErrorAction SilentlyContinue | Measure-Object | Select-Object -ExpandProperty Count",
	}

	result, err := w.executor.Run(ctx, command, args, executor.ExecOpts{})
	if err != nil {
		return false, err
	}

	count := strings.TrimSpace(result.Stdout)
	return count != "0" && count != "", nil
}

// StartService starts a service using PowerShell
func (w *WindowsServiceDetector) StartService(ctx context.Context, serviceName string) error {
	cmd := fmt.Sprintf("Start-Service -Name '%s'", serviceName)
	result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// StopService stops a service using PowerShell
func (w *WindowsServiceDetector) StopService(ctx context.Context, serviceName string) error {
	cmd := fmt.Sprintf("Stop-Service -Name '%s'", serviceName)
	result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to stop service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// RestartService restarts a service using PowerShell
func (w *WindowsServiceDetector) RestartService(ctx context.Context, serviceName string) error {
	cmd := fmt.Sprintf("Restart-Service -Name '%s'", serviceName)
	result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to restart service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// DisableService disables a service from starting automatically on system startup
func (w *WindowsServiceDetector) DisableService(ctx context.Context, serviceName string) error {
	cmd := fmt.Sprintf("Set-Service -Name '%s' -StartupType Disabled", serviceName)
	result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to disable service %s: %s", serviceName, result.Stderr)
	}
	return nil
}

// IsServiceEnabled checks if a service is enabled for automatic startup
func (w *WindowsServiceDetector) IsServiceEnabled(ctx context.Context, serviceName string) (bool, error) {
	cmd := fmt.Sprintf("(Get-Service -Name '%s').StartType", serviceName)
	result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check if service %s is enabled: %w", serviceName, err)
	}

	// PowerShell returns "Automatic" for enabled services
	enabled := strings.TrimSpace(result.Stdout) == "Automatic"
	return enabled, nil
}

// SetServiceStartup sets whether a service should start on system startup
func (w *WindowsServiceDetector) SetServiceStartup(ctx context.Context, serviceName string, enabled bool) error {
	if enabled {
		cmd := fmt.Sprintf("Set-Service -Name '%s' -StartupType Automatic", serviceName)
		result, err := w.executor.Run(ctx, "powershell", []string{"-Command", cmd}, executor.ExecOpts{})
		if err != nil {
			return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("failed to enable service %s: %s", serviceName, result.Stderr)
		}
		return nil
	}
	return w.DisableService(ctx, serviceName)
}

// GetServicesForPackage returns service names associated with a package
func (w *WindowsServiceDetector) GetServicesForPackage(packageName string) ([]string, error) {
	return w.mapping.GetServicesForPackage(packageName), nil
}

// GetPackageForService returns the package name associated with a service
func (w *WindowsServiceDetector) GetPackageForService(serviceName string) (string, error) {
	return w.mapping.GetPackageForService(serviceName), nil
}

// Ensure WindowsServiceDetector implements ServiceManager interface
var _ ServiceManager = (*WindowsServiceDetector)(nil)
