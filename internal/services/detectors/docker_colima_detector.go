package detectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jamesainslie/terraform-package/internal/services"
)

// DockerColimaDetector detects Docker → Colima dependency relationships
type DockerColimaDetector struct {
	name string
}

// NewDockerColimaDetector creates a new Docker-Colima dependency detector
func NewDockerColimaDetector() *DockerColimaDetector {
	return &DockerColimaDetector{
		name: "docker-colima",
	}
}

// GetName returns the detector name
func (d *DockerColimaDetector) GetName() string {
	return d.name
}

// DetectDependencies detects Docker → Colima dependencies
func (d *DockerColimaDetector) DetectDependencies(ctx context.Context, serviceName string) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// Only detect dependencies for Docker service
	if strings.ToLower(serviceName) != "docker" {
		return dependencies, nil
	}

	// Check if Docker context is configured for Colima
	colimaDependency, err := d.detectColimaDependency(ctx)
	if err != nil {
		return dependencies, nil // Not an error, just no dependency
	}

	if colimaDependency != nil {
		dependencies = append(dependencies, *colimaDependency)
	}

	return dependencies, nil
}

// detectColimaDependency detects if Docker depends on Colima
func (d *DockerColimaDetector) detectColimaDependency(ctx context.Context) (*services.ServiceDependency, error) {
	// Check if Docker context is configured for Colima
	dockerContext, err := d.getDockerContext(ctx)
	if err != nil {
		return nil, err
	}

	// Check if the context points to Colima
	if !d.isColimaContext(dockerContext) {
		return nil, nil // No Colima dependency
	}

	// Check if Colima is available
	colimaAvailable, err := d.isColimaAvailable(ctx)
	if err != nil || !colimaAvailable {
		return nil, nil // Colima not available
	}

	// Create dependency relationship
	dependency := &services.ServiceDependency{
		SourceService:  "docker",
		TargetService:  "colima",
		DependencyType: services.DependencyTypeProxy,
		StartupOrder:   1, // Colima must start before Docker
		ProxyConfig: &services.ProxyConfig{
			ProxyType:      "socket",
			ProxyEndpoint:  d.getDockerSocketPath(),
			TargetEndpoint: d.getColimaSocketPath(),
			Timeout:        30 * time.Second,
			Metadata: map[string]string{
				"context": dockerContext,
				"type":    "docker-colima-proxy",
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
			"context":     dockerContext,
			"socket_path": d.getDockerSocketPath(),
			"colima_path": d.getColimaSocketPath(),
		},
	}

	return dependency, nil
}

// getDockerContext gets the current Docker context
func (d *DockerColimaDetector) getDockerContext(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// isColimaContext checks if the Docker context is configured for Colima
func (d *DockerColimaDetector) isColimaContext(context string) bool {
	// Check for common Colima context names
	colimaContexts := []string{
		"colima-default",
		"colima",
		"default",
	}

	for _, colimaContext := range colimaContexts {
		if strings.Contains(strings.ToLower(context), strings.ToLower(colimaContext)) {
			return true
		}
	}

	return false
}

// isColimaAvailable checks if Colima is available and running
func (d *DockerColimaDetector) isColimaAvailable(ctx context.Context) (bool, error) {
	// Check if colima command exists
	_, err := exec.LookPath("colima")
	if err != nil {
		return false, nil // Colima not installed
	}

	// Check if Colima is running
	cmd := exec.CommandContext(ctx, "colima", "status")
	err = cmd.Run()
	return err == nil, nil
}

// getDockerSocketPath gets the Docker socket path
func (d *DockerColimaDetector) getDockerSocketPath() string {
	// Default Docker socket path
	return "/var/run/docker.sock"
}

// getColimaSocketPath gets the Colima Docker socket path
func (d *DockerColimaDetector) getColimaSocketPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/colima-docker.sock" // Fallback
	}
	return filepath.Join(homeDir, ".colima", "default", "docker.sock")
}

// DetectColimaDependencies detects if Colima has any dependencies
func (d *DockerColimaDetector) DetectColimaDependencies(ctx context.Context) ([]services.ServiceDependency, error) {
	var dependencies []services.ServiceDependency

	// Colima typically doesn't have service dependencies
	// It's a standalone VM manager
	return dependencies, nil
}

// ValidateDockerColimaSetup validates the Docker-Colima setup
func (d *DockerColimaDetector) ValidateDockerColimaSetup(ctx context.Context) error {
	// Check if Docker context is properly configured
	context, err := d.getDockerContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Docker context: %w", err)
	}

	if !d.isColimaContext(context) {
		return fmt.Errorf("docker context '%s' is not configured for Colima", context)
	}

	// Check if Colima is running
	colimaAvailable, err := d.isColimaAvailable(ctx)
	if err != nil {
		return fmt.Errorf("failed to check Colima availability: %w", err)
	}

	if !colimaAvailable {
		return fmt.Errorf("colima is not running or not available")
	}

	// Check if Docker can connect to Colima
	cmd := exec.CommandContext(ctx, "docker", "version")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("docker cannot connect to Colima: %w", err)
	}

	return nil
}
