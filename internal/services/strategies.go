package services

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if d.commands == nil || len(d.commands.Start) == 0 {
		return fmt.Errorf("no start command configured for service %s", serviceName)
	}

	result, err := d.executor.Run(ctx, d.commands.Start[0], d.commands.Start[1:], executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to start service %s with direct command: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("direct command start failed for %s: %s", serviceName, result.Stderr)
	}
	return nil
}

func (d *DirectCommandStrategy) StopService(ctx context.Context, serviceName string) error {
	if d.commands == nil || len(d.commands.Stop) == 0 {
		return fmt.Errorf("no stop command configured for service %s", serviceName)
	}

	result, err := d.executor.Run(ctx, d.commands.Stop[0], d.commands.Stop[1:], executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to stop service %s with direct command: %w", serviceName, err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("direct command stop failed for %s: %s", serviceName, result.Stderr)
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
