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
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jamesainslie/terraform-package/internal/executor"
)

// DefaultHealthChecker implements the HealthChecker interface
type DefaultHealthChecker struct {
	executor executor.Executor
	client   *http.Client
}

// NewDefaultHealthChecker creates a new health checker
func NewDefaultHealthChecker(executor executor.Executor) *DefaultHealthChecker {
	return &DefaultHealthChecker{
		executor: executor,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckCommand performs a command-based health check
func (h *DefaultHealthChecker) CheckCommand(ctx context.Context, command string, timeout time.Duration) (*HealthResult, error) {
	start := time.Now()

	// Create context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse command into command and args
	parts := parseCommand(command)
	if len(parts) == 0 {
		return &HealthResult{
			Healthy:      false,
			Error:        "empty command",
			ResponseTime: time.Since(start),
		}, nil
	}

	cmd := parts[0]
	args := parts[1:]

	// Execute command
	result, err := h.executor.Run(ctxTimeout, cmd, args, executor.ExecOpts{})
	responseTime := time.Since(start)

	if err != nil {
		return &HealthResult{
			Healthy:      false,
			Error:        fmt.Sprintf("command failed: %v", err),
			ResponseTime: responseTime,
			Metadata: map[string]interface{}{
				"command": command,
				"output":  result.Stdout,
			},
		}, nil
	}

	return &HealthResult{
		Healthy:      true,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"command": command,
			"output":  result.Stdout,
		},
	}, nil
}

// CheckHTTP performs an HTTP-based health check
func (h *DefaultHealthChecker) CheckHTTP(ctx context.Context, endpoint string, expectedStatus int, timeout time.Duration) (*HealthResult, error) {
	start := time.Now()

	// Create context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctxTimeout, "GET", endpoint, nil)
	if err != nil {
		return &HealthResult{
			Healthy:      false,
			Error:        fmt.Sprintf("failed to create request: %v", err),
			ResponseTime: time.Since(start),
		}, nil
	}

	// Set reasonable timeout on client
	client := &http.Client{
		Timeout: timeout,
	}

	// Perform request
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return &HealthResult{
			Healthy:      false,
			Error:        fmt.Sprintf("HTTP request failed: %v", err),
			ResponseTime: responseTime,
			Metadata: map[string]interface{}{
				"endpoint": endpoint,
			},
		}, nil
	}
	defer resp.Body.Close()

	// Check status code
	healthy := resp.StatusCode == expectedStatus
	var errorMsg string
	if !healthy {
		errorMsg = fmt.Sprintf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	return &HealthResult{
		Healthy:      healthy,
		Error:        errorMsg,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"endpoint":        endpoint,
			"status_code":     resp.StatusCode,
			"expected_status": expectedStatus,
		},
	}, nil
}

// CheckTCP performs a TCP connection health check
func (h *DefaultHealthChecker) CheckTCP(ctx context.Context, host string, port int, timeout time.Duration) (*HealthResult, error) {
	start := time.Now()

	// Create context with timeout
	ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Attempt TCP connection
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := dialer.DialContext(ctxTimeout, "tcp", address)
	responseTime := time.Since(start)

	if err != nil {
		return &HealthResult{
			Healthy:      false,
			Error:        fmt.Sprintf("TCP connection failed: %v", err),
			ResponseTime: responseTime,
			Metadata: map[string]interface{}{
				"host": host,
				"port": port,
			},
		}, nil
	}

	// Close connection immediately
	_ = conn.Close()

	return &HealthResult{
		Healthy:      true,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"host": host,
			"port": port,
		},
	}, nil
}

// CheckMultiple performs multiple health checks concurrently
func (h *DefaultHealthChecker) CheckMultiple(ctx context.Context, checks []HealthCheck) map[string]*HealthResult {
	results := make(map[string]*HealthResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, check := range checks {
		wg.Add(1)
		go func(c HealthCheck) {
			defer wg.Done()

			var result *HealthResult
			var err error

			switch c.Type {
			case HealthCheckTypeCommand:
				result, err = h.CheckCommand(ctx, c.Command, c.Timeout)
			case HealthCheckTypeHTTP:
				result, err = h.CheckHTTP(ctx, c.Endpoint, 200, c.Timeout) // Default to 200
			case HealthCheckTypeTCP:
				result, err = h.CheckTCP(ctx, c.Host, c.Port, c.Timeout)
			default:
				result = &HealthResult{
					Healthy: false,
					Error:   fmt.Sprintf("unknown health check type: %s", c.Type),
				}
			}

			if err != nil {
				result = &HealthResult{
					Healthy: false,
					Error:   err.Error(),
				}
			}

			mu.Lock()
			results[c.ServiceName] = result
			mu.Unlock()
		}(check)
	}

	wg.Wait()
	return results
}

// parseCommand parses a command string into command and arguments
// This is a simple implementation that splits on whitespace
// For production use, consider using shlex or similar for proper shell parsing
func parseCommand(command string) []string {
	// Simple whitespace splitting - could be enhanced with proper shell parsing
	parts := []string{}
	current := ""
	inQuotes := false

	for _, r := range command {
		switch r {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				current += string(r)
			} else {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			}
		default:
			current += string(r)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// Ensure DefaultHealthChecker implements HealthChecker interface
var _ HealthChecker = (*DefaultHealthChecker)(nil)
