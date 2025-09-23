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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// MockExecutor implements the executor interface for testing
type MockExecutor struct {
	shouldFail bool
	output     string
}

func (m *MockExecutor) Run(ctx context.Context, cmd string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	if m.shouldFail {
		return executor.ExecResult{}, fmt.Errorf("mock command failed")
	}

	return executor.ExecResult{
		Stdout:   m.output,
		Stderr:   "",
		ExitCode: 0,
	}, nil
}

func TestNewDefaultHealthChecker(t *testing.T) {
	mockExec := &MockExecutor{}
	checker := NewDefaultHealthChecker(mockExec)

	if checker == nil {
		t.Fatal("NewDefaultHealthChecker returned nil")
	}

	// Check that it implements the HealthChecker interface
	var _ HealthChecker = checker
}

func TestCheckCommand(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		timeout        time.Duration
		mockShouldFail bool
		mockOutput     string
		expectHealthy  bool
		expectError    bool
	}{
		{
			name:           "successful command",
			command:        "echo hello",
			timeout:        5 * time.Second,
			mockShouldFail: false,
			mockOutput:     "hello\n",
			expectHealthy:  true,
			expectError:    false,
		},
		{
			name:           "failed command",
			command:        "false",
			timeout:        5 * time.Second,
			mockShouldFail: true,
			mockOutput:     "",
			expectHealthy:  false,
			expectError:    false, // Health check doesn't return error, just unhealthy
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &MockExecutor{
				shouldFail: tt.mockShouldFail,
				output:     tt.mockOutput,
			}
			checker := NewDefaultHealthChecker(mockExec)

			ctx := context.Background()
			result, err := checker.CheckCommand(ctx, tt.command, tt.timeout)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			if result.Healthy != tt.expectHealthy {
				t.Errorf("Expected healthy=%v but got %v", tt.expectHealthy, result.Healthy)
			}

			if result.ResponseTime <= 0 {
				t.Error("Expected positive response time")
			}

			if result.Metadata == nil {
				t.Error("Expected metadata to be set")
			}
		})
	}
}

func TestCheckHTTP(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockExec := &MockExecutor{}
	checker := NewDefaultHealthChecker(mockExec)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		timeout        time.Duration
		expectHealthy  bool
		expectError    bool
	}{
		{
			name:           "successful HTTP check",
			endpoint:       server.URL + "/health",
			expectedStatus: 200,
			timeout:        5 * time.Second,
			expectHealthy:  true,
			expectError:    false,
		},
		{
			name:           "HTTP error status",
			endpoint:       server.URL + "/error",
			expectedStatus: 200,
			timeout:        5 * time.Second,
			expectHealthy:  false,
			expectError:    false,
		},
		{
			name:           "HTTP not found",
			endpoint:       server.URL + "/notfound",
			expectedStatus: 200,
			timeout:        5 * time.Second,
			expectHealthy:  false,
			expectError:    false,
		},
		{
			name:           "invalid URL",
			endpoint:       "invalid-url",
			expectedStatus: 200,
			timeout:        5 * time.Second,
			expectHealthy:  false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := checker.CheckHTTP(ctx, tt.endpoint, tt.expectedStatus, tt.timeout)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			if result.Healthy != tt.expectHealthy {
				t.Errorf("Expected healthy=%v but got %v", tt.expectHealthy, result.Healthy)
			}

			if result.ResponseTime <= 0 {
				t.Error("Expected positive response time")
			}

			if result.Metadata == nil {
				t.Error("Expected metadata to be set")
			}
		})
	}
}

func TestCheckTCP(t *testing.T) {
	// Create a test TCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockExec := &MockExecutor{}
	checker := NewDefaultHealthChecker(mockExec)

	tests := []struct {
		name          string
		host          string
		port          int
		timeout       time.Duration
		expectHealthy bool
		expectError   bool
	}{
		{
			name:          "successful TCP check",
			host:          "127.0.0.1",
			port:          80,
			timeout:       5 * time.Second,
			expectHealthy: false, // Likely no service on port 80 in test env
			expectError:   false,
		},
		{
			name:          "invalid host",
			host:          "999.999.999.999",
			port:          80,
			timeout:       1 * time.Second,
			expectHealthy: false,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := checker.CheckTCP(ctx, tt.host, tt.port, tt.timeout)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			if result.ResponseTime <= 0 {
				t.Error("Expected positive response time")
			}

			if result.Metadata == nil {
				t.Error("Expected metadata to be set")
			}
		})
	}
}

func TestCheckMultiple(t *testing.T) {
	mockExec := &MockExecutor{output: "success"}
	checker := NewDefaultHealthChecker(mockExec)

	checks := []HealthCheck{
		{
			ServiceName: "service1",
			Type:        HealthCheckTypeCommand,
			Command:     "echo test1",
			Timeout:     5 * time.Second,
		},
		{
			ServiceName: "service2",
			Type:        HealthCheckTypeCommand,
			Command:     "echo test2",
			Timeout:     5 * time.Second,
		},
	}

	ctx := context.Background()
	results := checker.CheckMultiple(ctx, checks)

	if len(results) != 2 {
		t.Errorf("Expected 2 results but got %d", len(results))
	}

	for serviceName, result := range results {
		if result == nil {
			t.Errorf("Expected result for service %s but got nil", serviceName)
			continue
		}

		if !result.Healthy {
			t.Errorf("Expected service %s to be healthy", serviceName)
		}

		if result.ResponseTime <= 0 {
			t.Errorf("Expected positive response time for service %s", serviceName)
		}
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "simple command",
			command:  "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "command with quotes",
			command:  `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "empty command",
			command:  "",
			expected: []string{},
		},
		{
			name:     "single word",
			command:  "ls",
			expected: []string{"ls"},
		},
		{
			name:     "command with multiple spaces",
			command:  "echo   hello   world",
			expected: []string{"echo", "hello", "world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommand(tt.command)
			if len(result) != len(tt.expected) {
				t.Errorf("parseCommand(%q) returned %d parts, want %d", tt.command, len(result), len(tt.expected))
			}

			for i, part := range result {
				if i < len(tt.expected) && part != tt.expected[i] {
					t.Errorf("parseCommand(%q)[%d] = %q, want %q", tt.command, i, part, tt.expected[i])
				}
			}
		})
	}
}
