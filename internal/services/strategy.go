package services

import (
	"context"
	"fmt"

	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// ServiceManagementStrategy represents the strategy for managing a service
type ServiceManagementStrategy string

const (
	StrategyAuto          ServiceManagementStrategy = "auto"
	StrategyBrewServices  ServiceManagementStrategy = "brew_services"
	StrategyDirectCommand ServiceManagementStrategy = "direct_command"
	StrategyLaunchd       ServiceManagementStrategy = "launchd"
	StrategyProcessOnly   ServiceManagementStrategy = "process_only"
)

// ServiceStrategy defines the interface for service management strategies
type ServiceStrategy interface {
	// StartService starts a service using this strategy
	StartService(ctx context.Context, serviceName string) error

	// StopService stops a service using this strategy
	StopService(ctx context.Context, serviceName string) error

	// RestartService restarts a service using this strategy
	RestartService(ctx context.Context, serviceName string) error

	// IsRunning checks if a service is running using this strategy
	IsRunning(ctx context.Context, serviceName string) (bool, error)

	// GetStrategyName returns the name of this strategy
	GetStrategyName() ServiceManagementStrategy
}

// ServiceLifecycleStrategy extends ServiceStrategy with health and status methods
// This ensures health checks use the same logic as management operations
type ServiceLifecycleStrategy interface {
	ServiceStrategy

	// HealthCheck performs a health check using strategy-specific logic
	HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error)

	// StatusCheck gets detailed status information using strategy-specific logic
	StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error)
}

// ServiceHealthInfo contains health check results with strategy context
type ServiceHealthInfo struct {
	Healthy  bool                      `json:"healthy"`
	Details  string                    `json:"details,omitempty"`
	Metrics  map[string]interface{}    `json:"metrics,omitempty"`
	Strategy ServiceManagementStrategy `json:"strategy"`
}

// ServiceStatusInfo contains detailed status information with strategy context
type ServiceStatusInfo struct {
	Running   bool                      `json:"running"`
	Enabled   bool                      `json:"enabled"`
	ProcessID string                    `json:"process_id,omitempty"`
	Details   string                    `json:"details,omitempty"`
	Metadata  map[string]interface{}    `json:"metadata,omitempty"`
	Strategy  ServiceManagementStrategy `json:"strategy"`
}

// CustomCommands represents custom commands for service management
type CustomCommands struct {
	Start   []string
	Stop    []string
	Restart []string
	Status  []string
}

// ServiceStrategyFactory creates service strategies based on configuration
type ServiceStrategyFactory struct {
	executor executor.Executor
}

// NewServiceStrategyFactory creates a new strategy factory
func NewServiceStrategyFactory(executor executor.Executor) *ServiceStrategyFactory {
	return &ServiceStrategyFactory{
		executor: executor,
	}
}

// CreateStrategy creates a service strategy based on the specified strategy type
func (f *ServiceStrategyFactory) CreateStrategy(strategy ServiceManagementStrategy, customCommands *CustomCommands) ServiceStrategy {
	switch strategy {
	case StrategyBrewServices:
		return &BrewServicesStrategy{executor: f.executor}
	case StrategyDirectCommand:
		return &DirectCommandStrategy{executor: f.executor, commands: customCommands}
	case StrategyLaunchd:
		return &LaunchdStrategy{executor: f.executor}
	case StrategyAuto:
		return &AutoStrategy{executor: f.executor, customCommands: customCommands}
	case StrategyProcessOnly:
		return &ProcessOnlyStrategy{executor: f.executor}
	default:
		// Default to auto strategy for unknown strategies
		return &AutoStrategy{executor: f.executor, customCommands: customCommands}
	}
}

// CreateLifecycleStrategy creates a service lifecycle strategy with health check capabilities
// This is the main entry point for creating strategies that support both management and health checks
func (f *ServiceStrategyFactory) CreateLifecycleStrategy(strategy ServiceManagementStrategy, customCommands *CustomCommands, serviceName string) ServiceLifecycleStrategy {
	// Get platform-specific defaults and service-specific overrides
	effectiveStrategy := f.resolveEffectiveStrategy(strategy, serviceName)
	effectiveCommands := f.resolveEffectiveCommands(customCommands, serviceName)

	// Create the appropriate lifecycle strategy
	switch effectiveStrategy {
	case StrategyBrewServices:
		return &BrewServicesLifecycleStrategy{
			BrewServicesStrategy: BrewServicesStrategy{executor: f.executor},
		}
	case StrategyDirectCommand:
		return &DirectCommandLifecycleStrategy{
			DirectCommandStrategy: DirectCommandStrategy{executor: f.executor, commands: effectiveCommands},
		}
	case StrategyLaunchd:
		return &LaunchdLifecycleStrategy{
			LaunchdStrategy: LaunchdStrategy{executor: f.executor},
		}
	case StrategyProcessOnly:
		return &ProcessOnlyLifecycleStrategy{
			ProcessOnlyStrategy: ProcessOnlyStrategy{executor: f.executor},
		}
	case StrategyAuto:
		return &AutoLifecycleStrategy{
			AutoStrategy: AutoStrategy{executor: f.executor, customCommands: effectiveCommands},
			factory:      f,
			serviceName:  serviceName,
		}
	default:
		// Default to auto lifecycle strategy
		return &AutoLifecycleStrategy{
			AutoStrategy: AutoStrategy{executor: f.executor, customCommands: effectiveCommands},
			factory:      f,
			serviceName:  serviceName,
		}
	}
}

// GetDefaultStrategyForService returns the recommended strategy for a given service
func (f *ServiceStrategyFactory) GetDefaultStrategyForService(serviceName string) ServiceManagementStrategy {
	// Check if service supports brew services
	if f.supportsBrewServices(serviceName) {
		return StrategyBrewServices
	}

	// Check if service has known direct commands
	if f.hasKnownDirectCommands(serviceName) {
		return StrategyDirectCommand
	}

	// Default to auto strategy
	return StrategyAuto
}

// supportsBrewServices checks if a service supports brew services management
func (f *ServiceStrategyFactory) supportsBrewServices(serviceName string) bool {
	// List of services known to support brew services
	brewServices := map[string]bool{
		"postgresql":     true,
		"mysql":          true,
		"redis":          true,
		"nginx":          true,
		"apache":         true,
		"mongodb":        true,
		"elasticsearch":  true,
		"kibana":         true,
		"logstash":       true,
		"rabbitmq":       true,
		"memcached":      true,
		"cassandra":      true,
		"influxdb":       true,
		"grafana":        true,
		"prometheus":     true,
		"consul":         true,
		"vault":          true,
		"nomad":          true,
		"terraform":      true,
		"ansible":        true,
		"vagrant":        true,
		"packer":         true,
		"docker":         true,
		"docker-compose": true,
		"kubectl":        true,
		"helm":           true,
		"minikube":       true,
		"kind":           true,
		"k3d":            true,
		"k9s":            true,
		"stern":          true,
		"kubectx":        true,
		"kubens":         true,
		"kustomize":      true,
		"skaffold":       true,
		"tilt":           true,
		"telepresence":   true,
		"istioctl":       true,
		"linkerd":        true,
	}

	return brewServices[serviceName]
}

// hasKnownDirectCommands checks if a service has known direct command patterns
func (f *ServiceStrategyFactory) hasKnownDirectCommands(serviceName string) bool {
	// List of services with known direct commands
	directCommandServices := map[string]bool{
		"colima":         true,
		"docker-desktop": true,
		"lima":           true,
		"podman":         true,
		"buildah":        true,
		"skopeo":         true,
		"nerdctl":        true,
		"containerd":     true,
		"runc":           true,
		"crun":           true,
		"gvisor":         true,
		"firecracker":    true,
		"qemu":           true,
		"virtualbox":     true,
		"vmware":         true,
		"parallels":      true,
		"hyperkit":       true,
		"xhyve":          true,
	}

	return directCommandServices[serviceName]
}

// GetDefaultCommandsForService returns default commands for a service if known
func (f *ServiceStrategyFactory) GetDefaultCommandsForService(serviceName string) *CustomCommands {
	// Known service commands
	serviceCommands := map[string]*CustomCommands{
		"colima": {
			Start:   []string{"colima", "start"},
			Stop:    []string{"colima", "stop"},
			Restart: []string{"colima", "restart"},
			Status:  []string{"colima", "status"},
		},
		"docker-desktop": {
			Start:   []string{"open", "-a", "Docker"},
			Stop:    []string{"killall", "Docker"},
			Restart: []string{"killall", "Docker", "&&", "open", "-a", "Docker"},
			Status:  []string{"pgrep", "-f", "Docker"},
		},
		"lima": {
			Start:   []string{"limactl", "start", "default"},
			Stop:    []string{"limactl", "stop", "default"},
			Restart: []string{"limactl", "stop", "default", "&&", "limactl", "start", "default"},
			Status:  []string{"limactl", "list"},
		},
		"podman": {
			Start:   []string{"podman", "machine", "start"},
			Stop:    []string{"podman", "machine", "stop"},
			Restart: []string{"podman", "machine", "restart"},
			Status:  []string{"podman", "machine", "list"},
		},
	}

	if commands, exists := serviceCommands[serviceName]; exists {
		return commands
	}

	// Default commands for unknown services
	return &CustomCommands{
		Start:   []string{serviceName, "start"},
		Stop:    []string{serviceName, "stop"},
		Restart: []string{serviceName, "restart"},
		Status:  []string{serviceName, "status"},
	}
}

// resolveEffectiveStrategy determines the actual strategy to use based on service and platform
func (f *ServiceStrategyFactory) resolveEffectiveStrategy(requestedStrategy ServiceManagementStrategy, serviceName string) ServiceManagementStrategy {
	// If a specific strategy was requested and it's not auto, use it
	if requestedStrategy != StrategyAuto {
		return requestedStrategy
	}

	// For auto strategy, determine the best strategy for this service
	return f.GetDefaultStrategyForService(serviceName)
}

// resolveEffectiveCommands determines the effective commands to use, with service-specific defaults
func (f *ServiceStrategyFactory) resolveEffectiveCommands(customCommands *CustomCommands, serviceName string) *CustomCommands {
	// If custom commands are provided, use them
	if customCommands != nil {
		return customCommands
	}

	// Otherwise, get default commands for the service
	return f.GetDefaultCommandsForService(serviceName)
}

// ValidateStrategy validates that a strategy is supported
func ValidateStrategy(strategy string) error {
	validStrategies := []ServiceManagementStrategy{
		StrategyAuto,
		StrategyBrewServices,
		StrategyDirectCommand,
		StrategyLaunchd,
		StrategyProcessOnly,
	}

	for _, validStrategy := range validStrategies {
		if string(validStrategy) == strategy {
			return nil
		}
	}

	return fmt.Errorf("invalid strategy '%s'. Valid strategies are: %v", strategy, validStrategies)
}
