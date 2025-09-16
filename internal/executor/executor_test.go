// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package executor

import (
	"context"
	"runtime"
	"strings"
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

func TestSystemExecutor_RunWithWorkDir(t *testing.T) {
	executor := NewSystemExecutor()
	ctx := context.Background()

	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	result, err := executor.Run(ctx, "pwd", []string{}, ExecOpts{
		WorkDir: tmpDir,
		Timeout: 5 * time.Second,
	})

	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Output should contain the temp directory path
	if !strings.Contains(result.Stdout, tmpDir) {
		t.Errorf("Expected stdout to contain %s, got %s", tmpDir, result.Stdout)
	}
}

func TestSystemExecutor_RunWithEnv(t *testing.T) {
	executor := NewSystemExecutor()
	ctx := context.Background()

	testVar := "TEST_VAR_12345"
	testValue := "test_value_67890"

	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/C", "echo %TEST_VAR_12345%"}
	} else {
		cmd = "sh"
		args = []string{"-c", "echo $TEST_VAR_12345"}
	}

	result, err := executor.Run(ctx, cmd, args, ExecOpts{
		Env:     []string{testVar + "=" + testValue},
		Timeout: 5 * time.Second,
	})

	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, testValue) {
		t.Errorf("Expected stdout to contain %s, got %s", testValue, result.Stdout)
	}
}

func TestSystemExecutor_RunCommandNotFound(t *testing.T) {
	executor := NewSystemExecutor()
	ctx := context.Background()

	result, err := executor.Run(ctx, "nonexistentcommand12345", []string{}, ExecOpts{
		Timeout: 5 * time.Second,
	})

	if err == nil {
		t.Error("Expected error for non-existent command, got nil")
	}

	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1 for command not found, got %d", result.ExitCode)
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

func TestSystemExecutor_PrepareCommand(t *testing.T) {
	executor := NewSystemExecutor()

	tests := []struct {
		name        string
		cmd         string
		args        []string
		opts        ExecOpts
		expectedCmd string
		skipOnOS    string
	}{
		{
			name:        "basic command without sudo",
			cmd:         "echo",
			args:        []string{"hello"},
			opts:        ExecOpts{UseSudo: false},
			expectedCmd: "echo",
		},
		{
			name:        "command with sudo on unix",
			cmd:         "apt-get",
			args:        []string{"update"},
			opts:        ExecOpts{UseSudo: true, NonInteractive: true},
			expectedCmd: "sudo",
			skipOnOS:    "windows",
		},
		{
			name:        "sudo ignored on windows",
			cmd:         "winget",
			args:        []string{"list"},
			opts:        ExecOpts{UseSudo: true},
			expectedCmd: "winget",
			skipOnOS:    "darwin,linux",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Skip test if not appropriate for current OS
			if test.skipOnOS != "" {
				skipOSes := strings.Split(test.skipOnOS, ",")
				for _, skipOS := range skipOSes {
					if runtime.GOOS == skipOS {
						t.Skipf("Skipping test on %s", runtime.GOOS)
						return
					}
				}
			}

			finalCmd, finalArgs := executor.prepareCommand(test.cmd, test.args, test.opts)

			if finalCmd != test.expectedCmd {
				t.Errorf("Expected command %s, got %s", test.expectedCmd, finalCmd)
			}

			// For sudo commands, verify arguments are properly structured
			if test.opts.UseSudo && runtime.GOOS != "windows" {
				if len(finalArgs) < 2 {
					t.Error("Expected sudo command to have at least 2 arguments")
				}
				if finalArgs[0] != "-n" {
					t.Errorf("Expected first sudo arg to be -n, got %s", finalArgs[0])
				}
			}
		})
	}
}

func TestSystemExecutor_TruncateOutput(t *testing.T) {
	executor := NewSystemExecutor()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short output unchanged",
			input:    "hello world",
			maxLen:   50,
			expected: "hello world",
		},
		{
			name:     "long output truncated",
			input:    "this is a very long output that should be truncated",
			maxLen:   10,
			expected: "this is a ... (truncated)",
		},
		{
			name:     "exact length unchanged",
			input:    "exactly",
			maxLen:   7,
			expected: "exactly",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := executor.truncateOutput(test.input, test.maxLen)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestDetectPrivilegeEscalation(t *testing.T) {
	ctx := context.Background()

	switch runtime.GOOS {
	case "windows":
		// On Windows, should return an error as it's not implemented
		canElevate, err := DetectPrivilegeEscalation(ctx)
		if err == nil {
			t.Error("Expected error on Windows, got nil")
		}
		if canElevate {
			t.Error("Expected false on Windows due to unimplemented detection")
		}

	case "darwin", "linux":
		// On Unix systems, test sudo availability
		canElevate, err := DetectPrivilegeEscalation(ctx)

		// Should not error if sudo is available
		if !IsCommandAvailable(ctx, "sudo") {
			if err == nil {
				t.Error("Expected error when sudo is not available")
			}
			if canElevate {
				t.Error("Expected false when sudo is not available")
			}
		} else {
			// If sudo is available, the result depends on sudo configuration
			// We can't guarantee the result, but should not panic
			t.Logf("Privilege escalation detection result: %v (error: %v)", canElevate, err)
		}

	default:
		canElevate, err := DetectPrivilegeEscalation(ctx)
		if err == nil {
			t.Error("Expected error for unsupported OS")
		}
		if canElevate {
			t.Error("Expected false for unsupported OS")
		}
	}
}

func TestSystemExecutor_RunWithSudo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping sudo test on Windows")
	}

	executor := NewSystemExecutor()

	// Test that sudo command is properly prepared (but don't actually run it)
	// This tests the prepareCommand logic without requiring actual sudo access
	finalCmd, finalArgs := executor.prepareCommand("echo", []string{"test"}, ExecOpts{
		UseSudo:        true,
		NonInteractive: true,
	})

	if finalCmd != "sudo" {
		t.Errorf("Expected sudo command, got %s", finalCmd)
	}

	expectedArgs := []string{"-n", "--", "echo", "test"}
	if len(finalArgs) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(finalArgs))
	}

	for i, expected := range expectedArgs {
		if i < len(finalArgs) && finalArgs[i] != expected {
			t.Errorf("Expected arg %d to be %s, got %s", i, expected, finalArgs[i])
		}
	}
}
