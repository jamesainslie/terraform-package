package detectors

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jamesainslie/terraform-provider-package/internal/services"
)

// ContainerRuntimeDetector detects container runtime dependencies
type ContainerRuntimeDetector struct {
	name string
}

// NewContainerRuntimeDetector creates a new container runtime dependency detector
func NewContainerRuntimeDetector() *ContainerRuntimeDetector {
	return &ContainerRuntimeDetector{
		name: "container-runtime",
	}
}

// GetName returns the detector name
func (d *ContainerRuntimeDetector) GetName() string {
	return d.name
}

// DetectDependencies detects container runtime dependencies
func (d *ContainerRuntimeDetector) DetectDependencies(ctx context.Context, serviceName string) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// Detect different container runtime patterns
	switch strings.ToLower(serviceName) {
	case "docker":
		deps, err := d.detectDockerDependencies(ctx)
		if err != nil {
			return dependencies, nil // Not an error, just no dependency
		}
		dependencies = append(dependencies, deps...)
	case "podman":
		deps, err := d.detectPodmanDependencies(ctx)
		if err != nil {
			return dependencies, nil
		}
		dependencies = append(dependencies, deps...)
	case "containerd":
		deps, err := d.detectContainerdDependencies(ctx)
		if err != nil {
			return dependencies, nil
		}
		dependencies = append(dependencies, deps...)
	}

	return dependencies, nil
}

// detectDockerDependencies detects Docker dependencies
func (d *ContainerRuntimeDetector) detectDockerDependencies(ctx context.Context) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// Check for Docker Desktop (macOS/Windows)
	if d.isDockerDesktop(ctx) {
		// Docker Desktop manages its own VM, no external dependencies
		return dependencies, nil
	}

	// Check for Docker with Colima
	if d.isDockerWithColima(ctx) {
		dependency := services.ServiceDependency{
			SourceService:  "docker",
			TargetService:  "colima",
			DependencyType: services.DependencyTypeProxy,
			StartupOrder:   1,
			ProxyConfig: &services.ProxyConfig{
				ProxyType:      "socket",
				ProxyEndpoint:  "/var/run/docker.sock",
				TargetEndpoint: d.getColimaSocketPath(),
				Timeout:        30 * time.Second,
				Metadata: map[string]string{
					"type": "docker-colima-proxy",
				},
			},
			HealthCheck: &services.DependencyHealthCheck{
				Type:     "command",
				Command:  "docker version",
				Timeout:  10 * time.Second,
				Retries:  3,
				Interval: 5 * time.Second,
			},
			Metadata: map[string]string{
				"detector":    d.name,
				"runtime":     "colima",
				"socket_path": d.getColimaSocketPath(),
			},
		}
		dependencies = append(dependencies, dependency)
	}

	// Check for Docker with Podman Machine
	if d.isDockerWithPodmanMachine(ctx) {
		dependency := services.ServiceDependency{
			SourceService:  "docker",
			TargetService:  "podman-machine",
			DependencyType: services.DependencyTypeProxy,
			StartupOrder:   1,
			ProxyConfig: &services.ProxyConfig{
				ProxyType:      "socket",
				ProxyEndpoint:  "/var/run/docker.sock",
				TargetEndpoint: d.getPodmanMachineSocketPath(),
				Timeout:        30 * time.Second,
				Metadata: map[string]string{
					"type": "docker-podman-machine-proxy",
				},
			},
			HealthCheck: &services.DependencyHealthCheck{
				Type:     "command",
				Command:  "docker version",
				Timeout:  10 * time.Second,
				Retries:  3,
				Interval: 5 * time.Second,
			},
			Metadata: map[string]string{
				"detector":    d.name,
				"runtime":     "podman-machine",
				"socket_path": d.getPodmanMachineSocketPath(),
			},
		}
		dependencies = append(dependencies, dependency)
	}

	return dependencies, nil
}

// detectPodmanDependencies detects Podman dependencies
func (d *ContainerRuntimeDetector) detectPodmanDependencies(ctx context.Context) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// Check for Podman with Podman Machine
	if d.isPodmanWithMachine(ctx) {
		dependency := services.ServiceDependency{
			SourceService:  "podman",
			TargetService:  "podman-machine",
			DependencyType: services.DependencyTypeProxy,
			StartupOrder:   1,
			ProxyConfig: &services.ProxyConfig{
				ProxyType:      "socket",
				ProxyEndpoint:  d.getPodmanSocketPath(),
				TargetEndpoint: d.getPodmanMachineSocketPath(),
				Timeout:        30 * time.Second,
				Metadata: map[string]string{
					"type": "podman-machine-proxy",
				},
			},
			HealthCheck: &services.DependencyHealthCheck{
				Type:     "command",
				Command:  "podman version",
				Timeout:  10 * time.Second,
				Retries:  3,
				Interval: 5 * time.Second,
			},
			Metadata: map[string]string{
				"detector":    d.name,
				"runtime":     "podman-machine",
				"socket_path": d.getPodmanMachineSocketPath(),
			},
		}
		dependencies = append(dependencies, dependency)
	}

	return dependencies, nil
}

// detectContainerdDependencies detects containerd dependencies
func (d *ContainerRuntimeDetector) detectContainerdDependencies(ctx context.Context) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// containerd typically runs as a system service
	// Check if it's managed by systemd
	if d.isSystemdService(ctx, "containerd") {
		dependency := services.ServiceDependency{
			SourceService:  "containerd",
			TargetService:  "systemd",
			DependencyType: services.DependencyTypeRequired,
			StartupOrder:   1,
			HealthCheck: &services.DependencyHealthCheck{
				Type:     "command",
				Command:  "systemctl is-active containerd",
				Timeout:  10 * time.Second,
				Retries:  3,
				Interval: 5 * time.Second,
			},
			Metadata: map[string]string{
				"detector": d.name,
				"manager":  "systemd",
			},
		}
		dependencies = append(dependencies, dependency)
	}

	return dependencies, nil
}

// isDockerDesktop checks if Docker Desktop is running
func (d *ContainerRuntimeDetector) isDockerDesktop(ctx context.Context) bool {
	// Check for Docker Desktop process
	cmd := exec.CommandContext(ctx, "ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for Docker Desktop process
	return strings.Contains(string(output), "Docker Desktop") ||
		strings.Contains(string(output), "com.docker.backend")
}

// isDockerWithColima checks if Docker is configured with Colima
func (d *ContainerRuntimeDetector) isDockerWithColima(ctx context.Context) bool {
	// Check Docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	context := strings.TrimSpace(string(output))
	return strings.Contains(strings.ToLower(context), "colima")
}

// isDockerWithPodmanMachine checks if Docker is configured with Podman Machine
func (d *ContainerRuntimeDetector) isDockerWithPodmanMachine(ctx context.Context) bool {
	// Check Docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	context := strings.TrimSpace(string(output))
	return strings.Contains(strings.ToLower(context), "podman")
}

// isPodmanWithMachine checks if Podman is configured with a machine
func (d *ContainerRuntimeDetector) isPodmanWithMachine(ctx context.Context) bool {
	// Check Podman machine list
	cmd := exec.CommandContext(ctx, "podman", "machine", "list")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for active machines
	return strings.Contains(string(output), "Currently running")
}

// isSystemdService checks if a service is managed by systemd
func (d *ContainerRuntimeDetector) isSystemdService(ctx context.Context, serviceName string) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", serviceName)
	err := cmd.Run()
	return err == nil
}

// getColimaSocketPath gets the Colima Docker socket path
func (d *ContainerRuntimeDetector) getColimaSocketPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/colima-docker.sock"
	}
	return filepath.Join(homeDir, ".colima", "default", "docker.sock")
}

// getPodmanMachineSocketPath gets the Podman Machine socket path
func (d *ContainerRuntimeDetector) getPodmanMachineSocketPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/podman-machine.sock"
	}
	return filepath.Join(homeDir, ".local", "share", "containers", "podman", "machine", "podman-machine-default", "podman.sock")
}

// getPodmanSocketPath gets the Podman socket path
func (d *ContainerRuntimeDetector) getPodmanSocketPath() string {
	return "/run/podman/podman.sock"
}
