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

// Package services provides service detection and health checking capabilities
package services

import (
	"context"
	"time"
)

// ServiceDetector defines the interface for detecting and checking services
type ServiceDetector interface {
	// IsRunning checks if a service is currently running
	IsRunning(ctx context.Context, serviceName string) (bool, error)

	// GetServiceInfo retrieves detailed information about a service
	GetServiceInfo(ctx context.Context, serviceName string) (*ServiceInfo, error)

	// GetAllServices retrieves information about all detectable services
	GetAllServices(ctx context.Context) (map[string]*ServiceInfo, error)

	// CheckHealth performs health checks on a service
	CheckHealth(ctx context.Context, serviceName string, config *HealthCheckConfig) (*HealthResult, error)
}

// ServiceInfo represents detailed information about a service
type ServiceInfo struct {
	Name        string            `json:"name"`
	Running     bool              `json:"running"`
	Healthy     bool              `json:"healthy"`
	Version     string            `json:"version,omitempty"`
	ProcessID   string            `json:"process_id,omitempty"`
	StartTime   *time.Time        `json:"start_time,omitempty"`
	ManagerType string            `json:"manager_type"` // launchd, systemd, brew, etc.
	Package     *PackageInfo      `json:"package,omitempty"`
	Ports       []int             `json:"ports,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PackageInfo represents package information for a service
type PackageInfo struct {
	Name    string `json:"name"`
	Manager string `json:"manager"`
	Version string `json:"version"`
}

// HealthCheckConfig defines configuration for health checks
type HealthCheckConfig struct {
	Command        string        `json:"command,omitempty"`
	HTTPEndpoint   string        `json:"http_endpoint,omitempty"`
	ExpectedStatus int           `json:"expected_status,omitempty"`
	TCPHost        string        `json:"tcp_host,omitempty"`
	TCPPort        int           `json:"tcp_port,omitempty"`
	Timeout        time.Duration `json:"timeout"`
	RetryCount     int           `json:"retry_count"`
	RetryInterval  time.Duration `json:"retry_interval"`
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Healthy      bool                   `json:"healthy"`
	ResponseTime time.Duration          `json:"response_time"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ServiceDetectionError represents errors that occur during service detection
type ServiceDetectionError struct {
	ServiceName string
	Platform    string
	Cause       error
	Suggestion  string
}

func (e *ServiceDetectionError) Error() string {
	suggestion := ""
	if e.Suggestion != "" {
		suggestion = ". Suggestion: " + e.Suggestion
	}
	return "failed to detect service " + e.ServiceName + " on " + e.Platform + ": " + e.Cause.Error() + suggestion
}

func (e *ServiceDetectionError) Unwrap() error {
	return e.Cause
}

// HealthChecker defines the interface for performing health checks
type HealthChecker interface {
	// CheckCommand performs a command-based health check
	CheckCommand(ctx context.Context, command string, timeout time.Duration) (*HealthResult, error)

	// CheckHTTP performs an HTTP-based health check
	CheckHTTP(ctx context.Context, endpoint string, expectedStatus int, timeout time.Duration) (*HealthResult, error)

	// CheckTCP performs a TCP connection health check
	CheckTCP(ctx context.Context, host string, port int, timeout time.Duration) (*HealthResult, error)

	// CheckMultiple performs multiple health checks concurrently
	CheckMultiple(ctx context.Context, checks []HealthCheck) map[string]*HealthResult
}

// HealthCheck represents a single health check configuration
type HealthCheck struct {
	ServiceName string
	Type        HealthCheckType
	Command     string
	Endpoint    string
	Host        string
	Port        int
	Timeout     time.Duration
}

// HealthCheckType defines the type of health check
type HealthCheckType string

const (
	HealthCheckTypeCommand HealthCheckType = "command"
	HealthCheckTypeHTTP    HealthCheckType = "http"
	HealthCheckTypeTCP     HealthCheckType = "tcp"
)

// Platform represents the operating system platform
type Platform string

const (
	PlatformDarwin  Platform = "darwin"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
)

// ServiceManager represents the type of service manager
type ServiceManager string

const (
	ServiceManagerLaunchd         ServiceManager = "launchd"
	ServiceManagerSystemd         ServiceManager = "systemd"
	ServiceManagerBrewServices    ServiceManager = "brew"
	ServiceManagerWindowsServices ServiceManager = "windows"
	ServiceManagerProcess         ServiceManager = "process"
)
