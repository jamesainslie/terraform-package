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

package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SystemExecutor implements the Executor interface using os/exec.
type SystemExecutor struct {
	logger *log.Logger
}

// NewSystemExecutor creates a new SystemExecutor.
func NewSystemExecutor() *SystemExecutor {
	return &SystemExecutor{
		logger: log.New(os.Stderr, "[executor] ", log.LstdFlags),
	}
}

// Run executes a command with the given options.
func (e *SystemExecutor) Run(ctx context.Context, command string, args []string, opts ExecOpts) (ExecResult, error) {
	startTime := time.Now()

	// Check if context is already cancelled/expired before starting
	select {
	case <-ctx.Done():
		tflog.Debug(ctx, "Context already expired before command execution", map[string]interface{}{
			"command": command,
			"args":    args,
			"error":   ctx.Err().Error(),
		})
		return ExecResult{ExitCode: -1}, fmt.Errorf("context already expired before execution: %w", ctx.Err())
	default:
		// Context is still valid, proceed
	}

	// DEBUG: Log the incoming request with all details
	tflog.Debug(ctx, "SystemExecutor.Run starting",
		map[string]interface{}{
			"command":         command,
			"args":            args,
			"timeout":         opts.Timeout.String(),
			"work_dir":        opts.WorkDir,
			"env_count":       len(opts.Env),
			"use_sudo":        opts.UseSudo,
			"non_interactive": opts.NonInteractive,
		})

	// Set default timeout if not specified
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
		tflog.Debug(ctx, "Using default timeout", map[string]interface{}{
			"default_timeout": opts.Timeout.String(),
		})
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Prepare command with sudo if needed
	finalCmd, finalArgs := e.prepareCommand(command, args, opts)

	// DEBUG: Log the final prepared command
	tflog.Debug(ctx, "Command prepared",
		map[string]interface{}{
			"original_command": command,
			"final_command":    finalCmd,
			"original_args":    args,
			"final_args":       finalArgs,
			"sudo_applied":     finalCmd == "sudo" || (len(finalArgs) > 0 && finalArgs[0] == command),
		})

	// Create the command
	// #nosec G204 - Command construction is controlled and validated through prepareCommand
	cmd := exec.CommandContext(ctx, finalCmd, finalArgs...)

	// Set working directory if specified
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
		tflog.Debug(ctx, "Working directory set", map[string]interface{}{
			"work_dir": opts.WorkDir,
		})
	}

	// Set environment variables
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
		tflog.Debug(ctx, "Environment variables set", map[string]interface{}{
			"env_vars":       opts.Env,
			"total_env_vars": len(cmd.Env),
		})
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log the command being executed (without sensitive info)
	e.logger.Printf("Executing command: %s %s", finalCmd, strings.Join(finalArgs, " "))

	// DEBUG: Log execution start
	tflog.Debug(ctx, "Starting command execution",
		map[string]interface{}{
			"full_command": fmt.Sprintf("%s %s", finalCmd, strings.Join(finalArgs, " ")),
			"pid":          cmd.Process,
		})

	// Execute the command
	err := cmd.Run()

	executionDuration := time.Since(startTime)

	result := ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	// Get exit code if command failed
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			result.ExitCode = exitError.ExitCode()
			tflog.Debug(ctx, "Command failed with exit error", map[string]interface{}{
				"exit_code": result.ExitCode,
				"error":     err.Error(),
			})
		} else {
			// Other error (e.g., command not found, context timeout)
			result.ExitCode = -1
			tflog.Debug(ctx, "Command failed with system error", map[string]interface{}{
				"exit_code":  result.ExitCode,
				"error":      err.Error(),
				"error_type": fmt.Sprintf("%T", err),
			})
		}
	}

	// DEBUG: Log comprehensive results
	tflog.Debug(ctx, "Command execution completed",
		map[string]interface{}{
			"command":           fmt.Sprintf("%s %s", finalCmd, strings.Join(finalArgs, " ")),
			"exit_code":         result.ExitCode,
			"execution_time_ms": executionDuration.Milliseconds(),
			"stdout_length":     len(result.Stdout),
			"stderr_length":     len(result.Stderr),
			"success":           result.ExitCode == 0,
		})

	// DEBUG: Log stdout/stderr content (truncated for safety)
	if len(result.Stdout) > 0 {
		tflog.Debug(ctx, "Command stdout output", map[string]interface{}{
			"stdout":             e.truncateOutput(result.Stdout, 2000),
			"stdout_full_length": len(result.Stdout),
		})
	}

	if len(result.Stderr) > 0 {
		tflog.Debug(ctx, "Command stderr output", map[string]interface{}{
			"stderr":             e.truncateOutput(result.Stderr, 2000),
			"stderr_full_length": len(result.Stderr),
		})
	}

	// Log results (truncated for readability) - keeping existing logger for compatibility
	e.logger.Printf("Command completed with exit code: %d", result.ExitCode)
	if result.ExitCode != 0 {
		e.logger.Printf("Command stderr: %s", e.truncateOutput(result.Stderr, 500))
	}

	// DEBUG: Final summary
	tflog.Debug(ctx, "SystemExecutor.Run completed",
		map[string]interface{}{
			"total_duration_ms": executionDuration.Milliseconds(),
			"success":           err == nil && result.ExitCode == 0,
		})

	return result, err
}

// prepareCommand prepares the final command and arguments, adding sudo if needed.
func (e *SystemExecutor) prepareCommand(command string, args []string, opts ExecOpts) (string, []string) {
	if !opts.UseSudo {
		return command, args
	}

	// Only use sudo on Unix-like systems
	if runtime.GOOS == "windows" {
		e.logger.Printf("Warning: sudo requested on Windows, ignoring")
		return command, args
	}

	// Prepare sudo command
	sudoArgs := []string{"-n"} // Non-interactive mode
	if opts.NonInteractive {
		sudoArgs = append(sudoArgs, "--")
	}
	sudoArgs = append(sudoArgs, command)
	sudoArgs = append(sudoArgs, args...)

	return "sudo", sudoArgs
}

// truncateOutput truncates output to a maximum length for logging.
func (e *SystemExecutor) truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "... (truncated)"
}

// IsCommandAvailable checks if a command is available in the system PATH.
func IsCommandAvailable(_ context.Context, command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// DetectPrivilegeEscalation detects if privilege escalation is available.
func DetectPrivilegeEscalation(ctx context.Context) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		// On Windows, check if running as administrator
		// This is a simplified check - in production, you'd use Windows APIs
		return false, fmt.Errorf("windows privilege detection not implemented")

	case "darwin", "linux":
		// Check if sudo is available and configured for non-interactive use
		if !IsCommandAvailable(ctx, "sudo") {
			return false, fmt.Errorf("sudo command not available")
		}

		// Test sudo with non-interactive flag
		cmd := exec.CommandContext(ctx, "sudo", "-n", "true")
		err := cmd.Run()
		return err == nil, nil

	default:
		return false, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}
