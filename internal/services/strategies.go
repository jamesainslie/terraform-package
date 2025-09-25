package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamesainslie/terraform-package/internal/executor"
)

// BrewServicesStrategy manages services using brew services
type BrewServicesStrategy struct {
	executor executor.Executor
}

func (b *BrewServicesStrategy) GetStrategyName() ServiceManagementStrategy {
	return StrategyBrewServices
}

func (b *BrewServicesStrategy) StartService(ctx context.Context, serviceName string) error {
	result, err := b.executor.Run(ctx, "brew", []string{"services", "start", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to start service %s with brew services: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("brew services start failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (b *BrewServicesStrategy) StopService(ctx context.Context, serviceName string) error {
	result, err := b.executor.Run(ctx, "brew", []string{"services", "stop", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s with brew services: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("brew services stop failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (b *BrewServicesStrategy) RestartService(ctx context.Context, serviceName string) error {
	result, err := b.executor.Run(ctx, "brew", []string{"services", "restart", serviceName}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to restart service %s with brew services: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("brew services restart failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (b *BrewServicesStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	result, err := b.executor.Run(ctx, "brew", []string{"services", "list", "--json"}, executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check brew services status: %w", err)
	}
	if result.ExitCode != 0 {
		return false, fmt.Errorf("brew services list failed: %s", result.Stderr)
	}

	// Parse JSON output to check if service is running
	// This is a simplified check - in practice, you'd parse the JSON properly
	return strings.Contains(result.Stdout, fmt.Sprintf(`"name":"%s"`, serviceName)) &&
		strings.Contains(result.Stdout, `"status":"started"`), nil
}

// DirectCommandStrategy manages services using direct commands
type DirectCommandStrategy struct {
	executor executor.Executor
	commands *CustomCommands
}

func (d *DirectCommandStrategy) GetStrategyName() ServiceManagementStrategy {
	return StrategyDirectCommand
}

func (d *DirectCommandStrategy) StartService(ctx context.Context, serviceName string) error {
	// DEBUG: Log service start request
	tflog.Debug(ctx, "DirectCommandStrategy.StartService starting",
		map[string]interface{}{
			"service_name": serviceName,
			"strategy":     "direct_command",
		})

	if d.commands == nil || len(d.commands.Start) == 0 {
		tflog.Debug(ctx, "No start command configured", map[string]interface{}{
			"service_name": serviceName,
		})
		return fmt.Errorf("no start command configured for service %s", serviceName)
	}

	// DEBUG: Log the configured start command
	tflog.Debug(ctx, "Start command configured", map[string]interface{}{
		"service_name":  serviceName,
		"start_command": d.commands.Start,
	})

	// Before attempting to start, check if service is already running
	// This provides idempotency for services that don't handle "already running" gracefully
	tflog.Debug(ctx, "Checking if service is already running", map[string]interface{}{
		"service_name": serviceName,
	})

	if running, err := d.IsRunning(ctx, serviceName); err == nil && running {
		// Service is already running, no action needed
		tflog.Debug(ctx, "Service already running, skipping start command", map[string]interface{}{
			"service_name": serviceName,
		})
		return nil
	} else if err != nil {
		tflog.Debug(ctx, "Could not determine service status, proceeding with start", map[string]interface{}{
			"service_name": serviceName,
			"error":        err.Error(),
		})
	} else {
		tflog.Debug(ctx, "Service not running, proceeding with start", map[string]interface{}{
			"service_name": serviceName,
		})
	}

	// DEBUG: Log command execution
	tflog.Debug(ctx, "Executing start command", map[string]interface{}{
		"service_name": serviceName,
		"command":      d.commands.Start[0],
		"args":         d.commands.Start[1:],
	})

	result, err := d.executor.Run(ctx, d.commands.Start[0], d.commands.Start[1:], executor.ExecOpts{})

	// DEBUG: Log execution results
	tflog.Debug(ctx, "Start command execution completed", map[string]interface{}{
		"service_name": serviceName,
		"exit_code":    result.ExitCode,
		"has_error":    err != nil,
		"stdout_len":   len(result.Stdout),
		"stderr_len":   len(result.Stderr),
	})

	if err != nil {
		tflog.Debug(ctx, "Start command failed with error", map[string]interface{}{
			"service_name": serviceName,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to start service %s with direct command: %w", serviceName, err)
	}

	if result.ExitCode != 0 {
		// Handle service-specific "already running" cases
		tflog.Debug(ctx, "Start command returned non-zero exit code, checking for already running patterns", map[string]interface{}{
			"service_name": serviceName,
			"exit_code":    result.ExitCode,
			"stderr":       result.Stderr,
		})

		if d.isAlreadyRunningError(serviceName, result.Stderr) {
			// Service reports it's already running - this is actually success
			tflog.Debug(ctx, "Detected 'already running' error pattern, treating as success", map[string]interface{}{
				"service_name": serviceName,
				"stderr":       result.Stderr,
			})
			return nil
		}

		tflog.Debug(ctx, "Start command failed with real error", map[string]interface{}{
			"service_name": serviceName,
			"exit_code":    result.ExitCode,
			"stderr":       result.Stderr,
		})

		return fmt.Errorf("direct command start failed for %s: exit code %d, stderr: %s",
			serviceName, result.ExitCode, result.Stderr)
	}

	tflog.Debug(ctx, "DirectCommandStrategy.StartService completed successfully", map[string]interface{}{
		"service_name": serviceName,
	})

	return nil
}

func (d *DirectCommandStrategy) StopService(ctx context.Context, serviceName string) error {
	if d.commands == nil || len(d.commands.Stop) == 0 {
		return fmt.Errorf("no stop command configured for service %s", serviceName)
	}

	// Before attempting to stop, check if service is already stopped
	// This provides idempotency for services that don't handle "already stopped" gracefully
	if running, err := d.IsRunning(ctx, serviceName); err == nil && !running {
		// Service is already stopped, no action needed
		return nil
	}

	result, err := d.executor.Run(ctx, d.commands.Stop[0], d.commands.Stop[1:], executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s with direct command: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		// Handle service-specific "already stopped" cases
		if d.isAlreadyStoppedError(serviceName, result.Stderr) {
			// Service reports it's already stopped - this is actually success
			return nil
		}
		return fmt.Errorf("direct command stop failed for %s: exit code %d, stderr: %s",
			serviceName, result.ExitCode, result.Stderr)
	}
	return nil
}

func (d *DirectCommandStrategy) RestartService(ctx context.Context, serviceName string) error {
	if d.commands == nil || len(d.commands.Restart) == 0 {
		// Fallback to stop then start
		if err := d.StopService(ctx, serviceName); err != nil {
			return err
		}
		time.Sleep(2 * time.Second) // Brief pause between stop and start
		return d.StartService(ctx, serviceName)
	}

	result, err := d.executor.Run(ctx, d.commands.Restart[0], d.commands.Restart[1:], executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to restart service %s with direct command: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("direct command restart failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (d *DirectCommandStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	if d.commands == nil || len(d.commands.Status) == 0 {
		// Fallback to process check
		return d.checkProcessName(ctx, serviceName)
	}

	result, err := d.executor.Run(ctx, d.commands.Status[0], d.commands.Status[1:], executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check service %s status with direct command: %w", serviceName, err)
	}

	// For most services, exit code 0 means running, non-zero means not running
	return result.ExitCode == 0, nil
}

func (d *DirectCommandStrategy) checkProcessName(ctx context.Context, serviceName string) (bool, error) {
	result, err := d.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		return false, nil // Process not found is not an error
	}
	return result.ExitCode == 0, nil
}

// isAlreadyRunningError checks if the error message indicates the service is already running
func (d *DirectCommandStrategy) isAlreadyRunningError(serviceName, stderr string) bool {
	// Convert to lowercase for case-insensitive matching
	lowerStderr := strings.ToLower(stderr)

	// Service-specific patterns for "already running" errors
	switch serviceName {
	case "colima":
		// Colima specific messages
		return strings.Contains(lowerStderr, "vm is already running") ||
			strings.Contains(lowerStderr, "already running") ||
			strings.Contains(lowerStderr, "vm already exists")
	case "docker":
		// Docker specific messages
		return strings.Contains(lowerStderr, "docker is already running") ||
			strings.Contains(lowerStderr, "daemon is already running")
	default:
		// Generic patterns that many services use
		return strings.Contains(lowerStderr, "already running") ||
			strings.Contains(lowerStderr, "already started") ||
			strings.Contains(lowerStderr, "service is running") ||
			strings.Contains(lowerStderr, "already active")
	}
}

// isAlreadyStoppedError checks if the error message indicates the service is already stopped
func (d *DirectCommandStrategy) isAlreadyStoppedError(serviceName, stderr string) bool {
	// Convert to lowercase for case-insensitive matching
	lowerStderr := strings.ToLower(stderr)

	// Service-specific patterns for "already stopped" errors
	switch serviceName {
	case "colima":
		// Colima specific messages
		return strings.Contains(lowerStderr, "vm is not running") ||
			strings.Contains(lowerStderr, "already stopped") ||
			strings.Contains(lowerStderr, "no vm found")
	case "docker":
		// Docker specific messages
		return strings.Contains(lowerStderr, "docker is not running") ||
			strings.Contains(lowerStderr, "daemon not running")
	default:
		// Generic patterns that many services use
		return strings.Contains(lowerStderr, "already stopped") ||
			strings.Contains(lowerStderr, "not running") ||
			strings.Contains(lowerStderr, "service is not running") ||
			strings.Contains(lowerStderr, "already inactive")
	}
}

// LaunchdStrategy manages services using launchd
type LaunchdStrategy struct {
	executor executor.Executor
}

func (l *LaunchdStrategy) GetStrategyName() ServiceManagementStrategy {
	return StrategyLaunchd
}

func (l *LaunchdStrategy) StartService(ctx context.Context, serviceName string) error {
	// Try to load and start the service
	result, err := l.executor.Run(ctx, "launchctl", []string{"load", "-w", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)}, executor.ExecOpts{})
	if err != nil {
		// Try user domain
		result, err = l.executor.Run(ctx, "launchctl", []string{"load", "-w", fmt.Sprintf("~/Library/LaunchAgents/%s.plist", serviceName)}, executor.ExecOpts{})
		if err != nil {
			return fmt.Errorf("failed to start service %s with launchd: %w", serviceName, err)
		}
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("launchd start failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (l *LaunchdStrategy) StopService(ctx context.Context, serviceName string) error {
	// Try to unload the service
	result, err := l.executor.Run(ctx, "launchctl", []string{"unload", "-w", fmt.Sprintf("/Library/LaunchDaemons/%s.plist", serviceName)}, executor.ExecOpts{})
	if err != nil {
		// Try user domain
		result, err = l.executor.Run(ctx, "launchctl", []string{"unload", "-w", fmt.Sprintf("~/Library/LaunchAgents/%s.plist", serviceName)}, executor.ExecOpts{})
		if err != nil {
			return fmt.Errorf("failed to stop service %s with launchd: %w", serviceName, err)
		}
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("launchd stop failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (l *LaunchdStrategy) RestartService(ctx context.Context, serviceName string) error {
	if err := l.StopService(ctx, serviceName); err != nil {
		return err
	}
	time.Sleep(2 * time.Second) // Brief pause between stop and start
	return l.StartService(ctx, serviceName)
}

func (l *LaunchdStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	result, err := l.executor.Run(ctx, "launchctl", []string{"list", serviceName}, executor.ExecOpts{})
	if err != nil {
		return false, fmt.Errorf("failed to check launchd service %s: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return false, nil // Service not found means not running
	}

	// Check if the service is loaded and running
	return !strings.Contains(result.Stdout, "-"), nil
}

// AutoStrategy tries multiple strategies in order of preference
type AutoStrategy struct {
	executor       executor.Executor
	customCommands *CustomCommands
}

func (a *AutoStrategy) GetStrategyName() ServiceManagementStrategy {
	return StrategyAuto
}

func (a *AutoStrategy) StartService(ctx context.Context, serviceName string) error {
	strategies := []ServiceStrategy{
		&BrewServicesStrategy{executor: a.executor},
		&DirectCommandStrategy{executor: a.executor, commands: a.customCommands},
		&LaunchdStrategy{executor: a.executor},
	}

	var lastErr error
	for _, strategy := range strategies {
		if err := strategy.StartService(ctx, serviceName); err == nil {
			return nil // Success
		} else {
			lastErr = err
		}
	}

	// Check if already running as final fallback
	if running, _ := a.IsRunning(ctx, serviceName); running {
		return nil
	}

	return fmt.Errorf("failed to start service %s with any strategy. Last error: %w", serviceName, lastErr)
}

func (a *AutoStrategy) StopService(ctx context.Context, serviceName string) error {
	strategies := []ServiceStrategy{
		&BrewServicesStrategy{executor: a.executor},
		&DirectCommandStrategy{executor: a.executor, commands: a.customCommands},
		&LaunchdStrategy{executor: a.executor},
	}

	var lastErr error
	for _, strategy := range strategies {
		if err := strategy.StopService(ctx, serviceName); err == nil {
			return nil // Success
		} else {
			lastErr = err
		}
	}

	// Check if already stopped as final fallback
	if running, _ := a.IsRunning(ctx, serviceName); !running {
		return nil
	}

	return fmt.Errorf("failed to stop service %s with any strategy. Last error: %w", serviceName, lastErr)
}

func (a *AutoStrategy) RestartService(ctx context.Context, serviceName string) error {
	strategies := []ServiceStrategy{
		&BrewServicesStrategy{executor: a.executor},
		&DirectCommandStrategy{executor: a.executor, commands: a.customCommands},
		&LaunchdStrategy{executor: a.executor},
	}

	var lastErr error
	for _, strategy := range strategies {
		if err := strategy.RestartService(ctx, serviceName); err == nil {
			return nil // Success
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("failed to restart service %s with any strategy. Last error: %w", serviceName, lastErr)
}

func (a *AutoStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	strategies := []ServiceStrategy{
		&BrewServicesStrategy{executor: a.executor},
		&DirectCommandStrategy{executor: a.executor, commands: a.customCommands},
		&LaunchdStrategy{executor: a.executor},
	}

	for _, strategy := range strategies {
		if running, err := strategy.IsRunning(ctx, serviceName); err == nil {
			return running, nil
		}
	}

	// Final fallback to process check
	result, err := a.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		return false, nil // Process not found is not an error
	}
	return result.ExitCode == 0, nil
}

// ProcessOnlyStrategy only checks if a process is running, doesn't manage it
type ProcessOnlyStrategy struct {
	executor executor.Executor
}

func (p *ProcessOnlyStrategy) GetStrategyName() ServiceManagementStrategy {
	return StrategyProcessOnly
}

func (p *ProcessOnlyStrategy) StartService(ctx context.Context, serviceName string) error {
	return fmt.Errorf("process-only strategy cannot start service %s - use a different strategy", serviceName)
}

func (p *ProcessOnlyStrategy) StopService(ctx context.Context, serviceName string) error {
	return fmt.Errorf("process-only strategy cannot stop service %s - use a different strategy", serviceName)
}

func (p *ProcessOnlyStrategy) RestartService(ctx context.Context, serviceName string) error {
	return fmt.Errorf("process-only strategy cannot restart service %s - use a different strategy", serviceName)
}

func (p *ProcessOnlyStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	result, err := p.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		return false, nil // Process not found is not an error
	}
	return result.ExitCode == 0, nil
}

// ============================================================================
// Lifecycle Strategy Implementations
// These extend the basic strategies with health check capabilities
// ============================================================================

// DirectCommandLifecycleStrategy extends DirectCommandStrategy with health checks
type DirectCommandLifecycleStrategy struct {
	DirectCommandStrategy
}

func (d *DirectCommandLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error) {
	tflog.Debug(ctx, "DirectCommandLifecycleStrategy.HealthCheck starting", map[string]interface{}{
		"service_name": serviceName,
		"strategy":     "direct_command",
	})

	if d.commands == nil || len(d.commands.Status) == 0 {
		// Fallback to process-based health check
		tflog.Debug(ctx, "No status command configured, using process check", map[string]interface{}{
			"service_name": serviceName,
		})
		
		running, err := d.checkProcessName(ctx, serviceName)
		return &ServiceHealthInfo{
			Healthy:  running,
			Details:  "Process-based health check (no status command configured)",
			Strategy: d.GetStrategyName(),
		}, err
	}

	tflog.Debug(ctx, "Executing status command for health check", map[string]interface{}{
		"service_name":    serviceName,
		"status_command":  d.commands.Status,
	})

	result, err := d.executor.Run(ctx, d.commands.Status[0], d.commands.Status[1:], executor.ExecOpts{})
	
	tflog.Debug(ctx, "Status command completed", map[string]interface{}{
		"service_name": serviceName,
		"exit_code":    result.ExitCode,
		"has_error":    err != nil,
		"stdout_len":   len(result.Stdout),
		"stderr_len":   len(result.Stderr),
	})

	if err != nil {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  fmt.Sprintf("Status command failed: %v", err),
			Strategy: d.GetStrategyName(),
		}, nil // Don't return error for health check failures
	}

	// For direct commands, exit code 0 typically means healthy
	healthy := result.ExitCode == 0
	details := fmt.Sprintf("Status command exit code: %d", result.ExitCode)
	
	// Add service-specific health parsing
	if healthy {
		details += d.parseHealthDetails(serviceName, result.Stdout, result.Stderr)
	} else {
		details += fmt.Sprintf(", stderr: %s", result.Stderr)
	}

	tflog.Debug(ctx, "DirectCommandLifecycleStrategy.HealthCheck completed", map[string]interface{}{
		"service_name": serviceName,
		"healthy":      healthy,
		"details":      details,
	})

	return &ServiceHealthInfo{
		Healthy:  healthy,
		Details:  details,
		Strategy: d.GetStrategyName(),
	}, nil
}

func (d *DirectCommandLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error) {
	// For direct command strategy, status check is similar to health check but with more details
	running, err := d.IsRunning(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if service is running: %w", err)
	}

	return &ServiceStatusInfo{
		Running:  running,
		Enabled:  false, // Direct command services typically don't have auto-start
		Details:  "Direct command service status",
		Strategy: d.GetStrategyName(),
	}, nil
}

// parseHealthDetails parses service-specific health information from command output
func (d *DirectCommandLifecycleStrategy) parseHealthDetails(serviceName, stdout, stderr string) string {
	switch serviceName {
	case "colima":
		if strings.Contains(stdout, "colima is running") {
			return " (Colima VM is running)"
		} else if strings.Contains(stdout, "colima is not running") {
			return " (Colima VM is not running)"
		}
		return " (Colima status checked)"
	case "docker":
		if strings.Contains(stdout, "Server:") {
			return " (Docker daemon is responding)"
		}
		return " (Docker status checked)"
	default:
		return " (Service status checked)"
	}
}

// BrewServicesLifecycleStrategy extends BrewServicesStrategy with health checks
type BrewServicesLifecycleStrategy struct {
	BrewServicesStrategy
}

func (b *BrewServicesLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error) {
	tflog.Debug(ctx, "BrewServicesLifecycleStrategy.HealthCheck starting", map[string]interface{}{
		"service_name": serviceName,
		"strategy":     "brew_services",
	})

	result, err := b.executor.Run(ctx, "brew", []string{"services", "list", "--json"}, executor.ExecOpts{})
	if err != nil {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  fmt.Sprintf("Failed to run brew services list: %v", err),
			Strategy: b.GetStrategyName(),
		}, nil
	}

	if result.ExitCode != 0 {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  fmt.Sprintf("brew services list failed: %s", result.Stderr),
			Strategy: b.GetStrategyName(),
		}, nil
	}

	// Parse JSON to determine if service is running and healthy
	healthy := strings.Contains(result.Stdout, fmt.Sprintf(`"name":"%s"`, serviceName)) &&
		strings.Contains(result.Stdout, `"status":"started"`)

	var details string
	if healthy {
		details = "Service is started via brew services"
	} else {
		details = "Service is not started via brew services"
	}

	tflog.Debug(ctx, "BrewServicesLifecycleStrategy.HealthCheck completed", map[string]interface{}{
		"service_name": serviceName,
		"healthy":      healthy,
		"details":      details,
	})

	return &ServiceHealthInfo{
		Healthy:  healthy,
		Details:  details,
		Strategy: b.GetStrategyName(),
	}, nil
}

func (b *BrewServicesLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error) {
	running, err := b.IsRunning(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if service is running: %w", err)
	}

	return &ServiceStatusInfo{
		Running:  running,
		Enabled:  running, // For brew services, running typically means enabled
		Details:  "Brew services status",
		Strategy: b.GetStrategyName(),
	}, nil
}

// LaunchdLifecycleStrategy extends LaunchdStrategy with health checks
type LaunchdLifecycleStrategy struct {
	LaunchdStrategy
}

func (l *LaunchdLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error) {
	tflog.Debug(ctx, "LaunchdLifecycleStrategy.HealthCheck starting", map[string]interface{}{
		"service_name": serviceName,
		"strategy":     "launchd",
	})

	result, err := l.executor.Run(ctx, "launchctl", []string{"list", serviceName}, executor.ExecOpts{})
	if err != nil {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  fmt.Sprintf("Failed to run launchctl list: %v", err),
			Strategy: l.GetStrategyName(),
		}, nil
	}

	if result.ExitCode != 0 {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  "Service not found in launchd",
			Strategy: l.GetStrategyName(),
		}, nil
	}

	// Check if the service is loaded and running (no "-" in the PID column)
	healthy := !strings.Contains(result.Stdout, "-")
	var details string
	if healthy {
		details = "Service is loaded and running in launchd"
	} else {
		details = "Service is loaded but not running in launchd"
	}

	tflog.Debug(ctx, "LaunchdLifecycleStrategy.HealthCheck completed", map[string]interface{}{
		"service_name": serviceName,
		"healthy":      healthy,
		"details":      details,
	})

	return &ServiceHealthInfo{
		Healthy:  healthy,
		Details:  details,
		Strategy: l.GetStrategyName(),
	}, nil
}

func (l *LaunchdLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error) {
	running, err := l.IsRunning(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if service is running: %w", err)
	}

	return &ServiceStatusInfo{
		Running:  running,
		Enabled:  true, // Launchd services are typically enabled if they're loaded
		Details:  "Launchd service status",
		Strategy: l.GetStrategyName(),
	}, nil
}

// ProcessOnlyLifecycleStrategy extends ProcessOnlyStrategy with health checks
type ProcessOnlyLifecycleStrategy struct {
	ProcessOnlyStrategy
}

func (p *ProcessOnlyLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error) {
	tflog.Debug(ctx, "ProcessOnlyLifecycleStrategy.HealthCheck starting", map[string]interface{}{
		"service_name": serviceName,
		"strategy":     "process_only",
	})

	result, err := p.executor.Run(ctx, "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
	if err != nil {
		return &ServiceHealthInfo{
			Healthy:  false,
			Details:  "No matching process found",
			Strategy: p.GetStrategyName(),
		}, nil
	}

	healthy := result.ExitCode == 0
	var details string
	if healthy {
		details = fmt.Sprintf("Process found (PIDs: %s)", strings.TrimSpace(result.Stdout))
	} else {
		details = "No matching process found"
	}

	tflog.Debug(ctx, "ProcessOnlyLifecycleStrategy.HealthCheck completed", map[string]interface{}{
		"service_name": serviceName,
		"healthy":      healthy,
		"details":      details,
	})

	return &ServiceHealthInfo{
		Healthy:  healthy,
		Details:  details,
		Strategy: p.GetStrategyName(),
	}, nil
}

func (p *ProcessOnlyLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error) {
	running, err := p.IsRunning(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if service is running: %w", err)
	}

	var processID string
	if running {
		result, _ := p.executor.Run(context.Background(), "pgrep", []string{"-f", serviceName}, executor.ExecOpts{})
		if result.ExitCode == 0 {
			processID = strings.TrimSpace(result.Stdout)
		}
	}

	return &ServiceStatusInfo{
		Running:   running,
		Enabled:   false, // Process-only services don't have auto-start
		ProcessID: processID,
		Details:   "Process-only service status",
		Strategy:  p.GetStrategyName(),
	}, nil
}

// AutoLifecycleStrategy extends AutoStrategy with health checks
type AutoLifecycleStrategy struct {
	AutoStrategy
	factory     *ServiceStrategyFactory
	serviceName string
}

func (a *AutoLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*ServiceHealthInfo, error) {
	tflog.Debug(ctx, "AutoLifecycleStrategy.HealthCheck starting", map[string]interface{}{
		"service_name": serviceName,
		"strategy":     "auto",
	})

	// Try each strategy in order until one succeeds
	strategies := []ServiceLifecycleStrategy{
		&BrewServicesLifecycleStrategy{BrewServicesStrategy: BrewServicesStrategy{executor: a.executor}},
		&DirectCommandLifecycleStrategy{DirectCommandStrategy: DirectCommandStrategy{executor: a.executor, commands: a.customCommands}},
		&LaunchdLifecycleStrategy{LaunchdStrategy: LaunchdStrategy{executor: a.executor}},
		&ProcessOnlyLifecycleStrategy{ProcessOnlyStrategy: ProcessOnlyStrategy{executor: a.executor}},
	}

	var lastErr error
	for _, strategy := range strategies {
		healthInfo, err := strategy.HealthCheck(ctx, serviceName)
		if err == nil && healthInfo.Healthy {
			// Found a strategy that reports the service as healthy
			healthInfo.Strategy = a.GetStrategyName() // Override to show this was auto-detected
			healthInfo.Details = fmt.Sprintf("Auto-detected via %s: %s", strategy.GetStrategyName(), healthInfo.Details)
			
			tflog.Debug(ctx, "AutoLifecycleStrategy.HealthCheck found healthy service", map[string]interface{}{
				"service_name":      serviceName,
				"detected_strategy": strategy.GetStrategyName(),
				"details":           healthInfo.Details,
			})
			
			return healthInfo, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	// No strategy found the service as healthy
	tflog.Debug(ctx, "AutoLifecycleStrategy.HealthCheck no strategy found service healthy", map[string]interface{}{
		"service_name": serviceName,
		"last_error":   lastErr,
	})

	return &ServiceHealthInfo{
		Healthy:  false,
		Details:  "Auto-detection: No strategy found service as healthy",
		Strategy: a.GetStrategyName(),
	}, nil
}

func (a *AutoLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*ServiceStatusInfo, error) {
	running, err := a.IsRunning(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if service is running: %w", err)
	}

	return &ServiceStatusInfo{
		Running:  running,
		Enabled:  false, // Auto strategy can't determine enabled status reliably
		Details:  "Auto-detected service status",
		Strategy: a.GetStrategyName(),
	}, nil
}
