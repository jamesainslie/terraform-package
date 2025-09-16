// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package executor

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestSystemExecutor_Run(t *testing.T) {
	executor := NewSystemExecutor()
	ctx := context.Background()

	// Test basic command execution
	result, err := executor.Run(ctx, "echo", []string{"hello"}, ExecOpts{
		Timeout: 10 * time.Second,
	})

	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout == "" {
		t.Error("Expected stdout output, got empty string")
	}
}

func TestSystemExecutor_RunWithTimeout(t *testing.T) {
	executor := NewSystemExecutor()
	ctx := context.Background()

	// Test command with very short timeout (should fail)
	_, err := executor.Run(ctx, "sleep", []string{"2"}, ExecOpts{
		Timeout: 100 * time.Millisecond,
	})

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestIsCommandAvailable(t *testing.T) {
	ctx := context.Background()

	// Test with a command that should exist on all systems
	var testCmd string
	switch runtime.GOOS {
	case "windows":
		testCmd = "cmd"
	default:
		testCmd = "sh"
	}

	if !IsCommandAvailable(ctx, testCmd) {
		t.Errorf("Command '%s' should be available", testCmd)
	}

	// Test with a command that should not exist
	if IsCommandAvailable(ctx, "nonexistentcommand12345") {
		t.Error("Nonexistent command should not be available")
	}
}
