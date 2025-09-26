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

package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/jamesainslie/terraform-package/internal/executor"
	"github.com/stretchr/testify/assert"
)

// MockExecutorForIdempotency provides more control for testing idempotency scenarios
type MockExecutorForIdempotency struct {
	// Commands to track what was called
	CalledCommands [][]string
	// Response map: command -> response
	Responses map[string]executor.ExecResult
	// Error map: command -> error
	Errors map[string]error
}

func NewMockExecutorForIdempotency() *MockExecutorForIdempotency {
	return &MockExecutorForIdempotency{
		CalledCommands: make([][]string, 0),
		Responses:      make(map[string]executor.ExecResult),
		Errors:         make(map[string]error),
	}
}

func (m *MockExecutorForIdempotency) SetResponse(command string, args []string, response executor.ExecResult, err error) {
	key := fmt.Sprintf("%s %v", command, args)
	m.Responses[key] = response
	if err != nil {
		m.Errors[key] = err
	}
}

func (m *MockExecutorForIdempotency) Run(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	// Track what was called
	fullCmd := append([]string{command}, args...)
	m.CalledCommands = append(m.CalledCommands, fullCmd)

	// Return predefined response
	key := fmt.Sprintf("%s %v", command, args)
	if response, exists := m.Responses[key]; exists {
		if err, hasErr := m.Errors[key]; hasErr {
			return response, err
		}
		return response, nil
	}

	// Default response
	return executor.ExecResult{ExitCode: 1, Stderr: "command not mocked"}, fmt.Errorf("command not mocked: %s %v", command, args)
}

func (m *MockExecutorForIdempotency) WasCalled(command string, args []string) bool {
	expected := append([]string{command}, args...)
	for _, called := range m.CalledCommands {
		if len(called) == len(expected) {
			match := true
			for i, arg := range expected {
				if called[i] != arg {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func (m *MockExecutorForIdempotency) CallCount() int {
	return len(m.CalledCommands)
}

func TestDirectCommandStrategy_StartService_AlreadyRunning(t *testing.T) {
	// Test that StartService skips start command if service is already running
	mockExec := NewMockExecutorForIdempotency()
	strategy := &DirectCommandStrategy{
		executor: mockExec,
		commands: &CustomCommands{
			Start:  []string{"colima", "start", "--cpu", "4"},
			Status: []string{"colima", "status"},
		},
	}

	// Mock status check to return "already running"
	mockExec.SetResponse("colima", []string{"status"},
		executor.ExecResult{ExitCode: 0, Stdout: "colima is running"}, nil)

	// This should succeed without attempting to start
	err := strategy.StartService(context.Background(), "colima")
	assert.NoError(t, err)

	// Verify status was checked
	assert.True(t, mockExec.WasCalled("colima", []string{"status"}))

	// Verify start command was NOT called since service is already running
	assert.False(t, mockExec.WasCalled("colima", []string{"start", "--cpu", "4"}))

	// Only one command should have been executed (status check)
	assert.Equal(t, 1, mockExec.CallCount())
}

func TestDirectCommandStrategy_StartService_ColimaAlreadyRunningError(t *testing.T) {
	// Test that StartService handles Colima's "VM is already running" error gracefully
	mockExec := NewMockExecutorForIdempotency()
	strategy := &DirectCommandStrategy{
		executor: mockExec,
		commands: &CustomCommands{
			Start:  []string{"colima", "start", "--cpu", "4"},
			Status: []string{"colima", "status"},
		},
	}

	// Mock status check to return "not running" initially
	mockExec.SetResponse("colima", []string{"status"},
		executor.ExecResult{ExitCode: 1, Stderr: "not running"}, nil)

	// Mock start command to fail with "VM is already running"
	mockExec.SetResponse("colima", []string{"start", "--cpu", "4"},
		executor.ExecResult{ExitCode: 1, Stderr: "VM is already running"}, nil)

	// This should succeed despite the error because it's a "VM already running" message
	err := strategy.StartService(context.Background(), "colima")
	assert.NoError(t, err)

	// Verify both commands were called
	assert.True(t, mockExec.WasCalled("colima", []string{"status"}))
	assert.True(t, mockExec.WasCalled("colima", []string{"start", "--cpu", "4"}))
	assert.Equal(t, 2, mockExec.CallCount())
}

func TestDirectCommandStrategy_IsAlreadyRunningError(t *testing.T) {
	strategy := &DirectCommandStrategy{}

	tests := []struct {
		name        string
		serviceName string
		stderr      string
		expected    bool
	}{
		{
			name:        "Colima VM already running",
			serviceName: "colima",
			stderr:      "VM is already running",
			expected:    true,
		},
		{
			name:        "Colima already running generic",
			serviceName: "colima",
			stderr:      "already running",
			expected:    true,
		},
		{
			name:        "Docker already running",
			serviceName: "docker",
			stderr:      "Docker is already running",
			expected:    true,
		},
		{
			name:        "Generic service already running",
			serviceName: "myservice",
			stderr:      "Service is already active",
			expected:    true,
		},
		{
			name:        "Real error message",
			serviceName: "colima",
			stderr:      "insufficient disk space",
			expected:    false,
		},
		{
			name:        "Empty stderr",
			serviceName: "colima",
			stderr:      "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.isAlreadyRunningError(tt.serviceName, tt.stderr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectCommandStrategy_IsAlreadyStoppedError(t *testing.T) {
	strategy := &DirectCommandStrategy{}

	tests := []struct {
		name        string
		serviceName string
		stderr      string
		expected    bool
	}{
		{
			name:        "Colima VM not running",
			serviceName: "colima",
			stderr:      "VM is not running",
			expected:    true,
		},
		{
			name:        "Colima already stopped",
			serviceName: "colima",
			stderr:      "already stopped",
			expected:    true,
		},
		{
			name:        "Docker not running",
			serviceName: "docker",
			stderr:      "Docker is not running",
			expected:    true,
		},
		{
			name:        "Generic service not running",
			serviceName: "myservice",
			stderr:      "Service is not running",
			expected:    true,
		},
		{
			name:        "Real error message",
			serviceName: "colima",
			stderr:      "permission denied",
			expected:    false,
		},
		{
			name:        "Empty stderr",
			serviceName: "colima",
			stderr:      "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.isAlreadyStoppedError(tt.serviceName, tt.stderr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
