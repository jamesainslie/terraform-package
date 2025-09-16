// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package executor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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
	// Set default timeout if not specified
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Minute
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Prepare command with sudo if needed
	finalCmd, finalArgs := e.prepareCommand(command, args, opts)

	// Create the command
	cmd := exec.CommandContext(ctx, finalCmd, finalArgs...)

	// Set working directory if specified
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Set environment variables
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log the command being executed (without sensitive info)
	e.logger.Printf("Executing command: %s %s", finalCmd, strings.Join(finalArgs, " "))

	// Execute the command
	err := cmd.Run()

	result := ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	// Get exit code if command failed
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			// Other error (e.g., command not found, context timeout)
			result.ExitCode = -1
		}
	}

	// Log results (truncated for readability)
	e.logger.Printf("Command completed with exit code: %d", result.ExitCode)
	if result.ExitCode != 0 {
		e.logger.Printf("Command stderr: %s", e.truncateOutput(result.Stderr, 500))
	}

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
func IsCommandAvailable(ctx context.Context, command string) bool {
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
