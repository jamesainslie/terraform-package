package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DependencyConfig represents configuration for service dependencies
type DependencyConfig struct {
	ServiceName    string                    `json:"service_name"`
	Strategy       ServiceManagementStrategy `json:"strategy"`
	CustomCommands *CustomCommands           `json:"custom_commands,omitempty"`
	HealthCheck    *DependencyHealthCheck    `json:"health_check,omitempty"`
	AutoDetect     bool                      `json:"auto_detect"`
	Metadata       map[string]string         `json:"metadata,omitempty"`
}

// DependencyConfigManager manages dependency configurations
type DependencyConfigManager struct {
	configs map[string]*DependencyConfig
}

// NewDependencyConfigManager creates a new dependency configuration manager
func NewDependencyConfigManager() *DependencyConfigManager {
	return &DependencyConfigManager{
		configs: make(map[string]*DependencyConfig),
	}
}

// RegisterConfig registers a dependency configuration
func (dcm *DependencyConfigManager) RegisterConfig(serviceName string, config *DependencyConfig) {
	dcm.configs[serviceName] = config
}

// GetConfig returns the configuration for a service
func (dcm *DependencyConfigManager) GetConfig(serviceName string) (*DependencyConfig, bool) {
	config, exists := dcm.configs[serviceName]
	return config, exists
}

// GetDefaultConfigs returns default configurations for common services
func (dcm *DependencyConfigManager) GetDefaultConfigs() map[string]*DependencyConfig {
	return map[string]*DependencyConfig{
		"colima": {
			ServiceName: "colima",
			Strategy:    StrategyDirectCommand,
			AutoDetect:  true,
			CustomCommands: &CustomCommands{
				Start:   []string{"colima", "start"},
				Stop:    []string{"colima", "stop"},
				Restart: []string{"colima", "restart"},
				Status:  []string{"colima", "status"},
			},
			HealthCheck: &DependencyHealthCheck{
				Type:     "command",
				Command:  "colima status",
				Timeout:  30 * time.Second,
				Retries:  3,
				Interval: 5 * time.Second,
			},
			Metadata: map[string]string{
				"description": "Container runtime for Docker on macOS",
				"category":    "container_runtime",
			},
		},
		"docker": {
			ServiceName: "docker",
			Strategy:    StrategyAuto,
			AutoDetect:  true,
			HealthCheck: &DependencyHealthCheck{
				Type:     "command",
				Command:  "docker version",
				Timeout:  10 * time.Second,
				Retries:  3,
				Interval: 2 * time.Second,
			},
			Metadata: map[string]string{
				"description": "Docker client for container management",
				"category":    "container_client",
			},
		},
	}
}

// DetectServiceConfiguration attempts to detect service configuration from the environment
func (dcm *DependencyConfigManager) DetectServiceConfiguration(ctx context.Context, serviceName string) (*DependencyConfig, error) {
	switch strings.ToLower(serviceName) {
	case "colima":
		return dcm.detectColimaConfiguration(ctx)
	case "docker":
		return dcm.detectDockerConfiguration(ctx)
	default:
		return nil, fmt.Errorf("no configuration detection available for service %s", serviceName)
	}
}

// detectColimaConfiguration detects Colima configuration from the environment
func (dcm *DependencyConfigManager) detectColimaConfiguration(ctx context.Context) (*DependencyConfig, error) {
	// Check if Colima is installed
	if _, err := exec.LookPath("colima"); err != nil {
		return nil, fmt.Errorf("colima not found in PATH")
	}

	// Try to get current Colima configuration
	cmd := exec.CommandContext(ctx, "colima", "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get colima status: %w", err)
	}

	// Parse the output to extract configuration
	config := &DependencyConfig{
		ServiceName: "colima",
		Strategy:    StrategyDirectCommand,
		AutoDetect:  true,
		CustomCommands: &CustomCommands{
			Start:   []string{"colima", "start"},
			Stop:    []string{"colima", "stop"},
			Restart: []string{"colima", "restart"},
			Status:  []string{"colima", "status"},
		},
		HealthCheck: &DependencyHealthCheck{
			Type:     "command",
			Command:  "colima status",
			Timeout:  30 * time.Second,
			Retries:  3,
			Interval: 5 * time.Second,
		},
		Metadata: map[string]string{
			"description": "Container runtime for Docker on macOS",
			"category":    "container_runtime",
		},
	}

	// Try to extract configuration from colima.yaml if it exists
	homeDir, err := os.UserHomeDir()
	if err == nil {
		colimaConfigPath := filepath.Join(homeDir, ".colima", "default", "colima.yaml")
		if _, err := os.Stat(colimaConfigPath); err == nil {
			config.Metadata["config_file"] = colimaConfigPath
		}
	}

	// Parse status output for additional metadata
	statusStr := string(output)
	if strings.Contains(statusStr, "runtime: docker") {
		config.Metadata["runtime"] = "docker"
	}
	if strings.Contains(statusStr, "arch:") {
		// Extract architecture
		lines := strings.Split(statusStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "arch:") {
				config.Metadata["architecture"] = strings.TrimSpace(strings.TrimPrefix(line, "arch:"))
				break
			}
		}
	}

	return config, nil
}

// detectDockerConfiguration detects Docker configuration from the environment
func (dcm *DependencyConfigManager) detectDockerConfiguration(ctx context.Context) (*DependencyConfig, error) {
	// Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH")
	}

	// Check Docker context
	cmd := exec.CommandContext(ctx, "docker", "context", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker context: %w", err)
	}

	contextName := strings.TrimSpace(string(output))

	config := &DependencyConfig{
		ServiceName: "docker",
		Strategy:    StrategyAuto,
		AutoDetect:  true,
		HealthCheck: &DependencyHealthCheck{
			Type:     "command",
			Command:  "docker version",
			Timeout:  10 * time.Second,
			Retries:  3,
			Interval: 2 * time.Second,
		},
		Metadata: map[string]string{
			"description": "Docker client for container management",
			"category":    "container_client",
			"context":     contextName,
		},
	}

	// Check if Docker context is Colima
	if contextName == "colima" || strings.Contains(contextName, "colima") {
		config.Metadata["runtime"] = "colima"
		config.Metadata["dependency"] = "colima"
	}

	return config, nil
}
