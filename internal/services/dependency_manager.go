package services

import (
	"context"
	"fmt"
	"time"
)

// ServiceDependency represents a dependency relationship between services
type ServiceDependency struct {
	SourceService  string                 `json:"source_service"`
	TargetService  string                 `json:"target_service"`
	DependencyType DependencyType         `json:"dependency_type"`
	ProxyConfig    *ProxyConfig           `json:"proxy_config,omitempty"`
	HealthCheck    *DependencyHealthCheck `json:"health_check,omitempty"`
	StartupOrder   int                    `json:"startup_order"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
}

// DependencyType defines the type of service dependency
type DependencyType string

const (
	DependencyTypeProxy     DependencyType = "proxy"     // Source service proxies to target service
	DependencyTypeForward   DependencyType = "forward"   // Source service forwards to target service
	DependencyTypeContainer DependencyType = "container" // Source service runs in target service's container
	DependencyTypeRequired  DependencyType = "required"  // Source service requires target service to be running
	DependencyTypeOptional  DependencyType = "optional"  // Source service can work without target service
)

// ProxyConfig defines configuration for proxy/forwarding relationships
type ProxyConfig struct {
	ProxyType       string            `json:"proxy_type"`      // "socket", "http", "tcp", "unix"
	ProxyEndpoint   string            `json:"proxy_endpoint"`  // Where the proxy listens
	TargetEndpoint  string            `json:"target_endpoint"` // Where the proxy forwards to
	HealthCheckPath string            `json:"health_check_path,omitempty"`
	HealthCheckPort int               `json:"health_check_port,omitempty"`
	Timeout         time.Duration     `json:"timeout,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// DependencyHealthCheck defines health checking for dependency relationships
type DependencyHealthCheck struct {
	Type       string        `json:"type"` // "command", "http", "tcp", "socket"
	Command    string        `json:"command,omitempty"`
	URL        string        `json:"url,omitempty"`
	Port       int           `json:"port,omitempty"`
	SocketPath string        `json:"socket_path,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
	Retries    int           `json:"retries,omitempty"`
	Interval   time.Duration `json:"interval,omitempty"`
}

// DependencyManager manages service dependencies and proxy relationships
type DependencyManager struct {
	dependencies map[string][]ServiceDependency
	detectors    []DependencyDetector
}

// DependencyDetector interface for detecting service dependencies
type DependencyDetector interface {
	DetectDependencies(ctx context.Context, serviceName string) ([]ServiceDependency, error)
	GetName() string
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{
		dependencies: make(map[string][]ServiceDependency),
		detectors:    []DependencyDetector{},
	}
}

// RegisterDetector registers a dependency detector
func (dm *DependencyManager) RegisterDetector(detector DependencyDetector) {
	dm.detectors = append(dm.detectors, detector)
}

// DetectDependencies detects all dependencies for a service
func (dm *DependencyManager) DetectDependencies(ctx context.Context, serviceName string) ([]ServiceDependency, error) {
	var allDependencies []ServiceDependency

	for _, detector := range dm.detectors {
		deps, err := detector.DetectDependencies(ctx, serviceName)
		if err != nil {
			// Log error but continue with other detectors
			continue
		}
		allDependencies = append(allDependencies, deps...)
	}

	dm.dependencies[serviceName] = allDependencies
	return allDependencies, nil
}

// GetDependencies returns cached dependencies for a service
func (dm *DependencyManager) GetDependencies(serviceName string) []ServiceDependency {
	return dm.dependencies[serviceName]
}

// GetDependencyChain returns the full dependency chain for a service
func (dm *DependencyManager) GetDependencyChain(ctx context.Context, serviceName string) ([]ServiceDependency, error) {
	var chain []ServiceDependency
	visited := make(map[string]bool)

	err := dm.buildDependencyChain(ctx, serviceName, &chain, visited)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// buildDependencyChain recursively builds the dependency chain
func (dm *DependencyManager) buildDependencyChain(ctx context.Context, serviceName string, chain *[]ServiceDependency, visited map[string]bool) error {
	if visited[serviceName] {
		return fmt.Errorf("circular dependency detected: %s", serviceName)
	}

	visited[serviceName] = true
	defer delete(visited, serviceName)

	deps, err := dm.DetectDependencies(ctx, serviceName)
	if err != nil {
		return err
	}

	for _, dep := range deps {
		*chain = append(*chain, dep)

		// Recursively build chain for target service
		if dep.TargetService != serviceName {
			err := dm.buildDependencyChain(ctx, dep.TargetService, chain, visited)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateDependencyChain validates that all dependencies can be satisfied
func (dm *DependencyManager) ValidateDependencyChain(ctx context.Context, serviceName string) error {
	chain, err := dm.GetDependencyChain(ctx, serviceName)
	if err != nil {
		return err
	}

	for _, dep := range chain {
		if dep.DependencyType == DependencyTypeRequired {
			// Check if target service is available
			// This would integrate with the service detection system
			// For now, we'll just validate the configuration
			if dep.TargetService == "" {
				return fmt.Errorf("required dependency %s has no target service", dep.SourceService)
			}
		}
	}

	return nil
}

// GetStartupOrder returns services in the correct startup order
func (dm *DependencyManager) GetStartupOrder(ctx context.Context, serviceName string) ([]string, error) {
	chain, err := dm.GetDependencyChain(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// Sort by startup order
	startupOrder := make(map[string]int)
	for _, dep := range chain {
		if dep.StartupOrder > 0 {
			startupOrder[dep.TargetService] = dep.StartupOrder
		}
	}

	// Simple topological sort based on startup order
	var ordered []string
	for service, order := range startupOrder {
		if order == 1 {
			ordered = append(ordered, service)
		}
	}

	return ordered, nil
}
