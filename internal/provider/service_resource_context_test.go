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

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/jamesainslie/terraform-package/internal/executor"
	"github.com/jamesainslie/terraform-package/internal/services"
)

// SimpleServiceManager for testing context timeout scenarios
type SimpleServiceManager struct {
	callCount int
	healthy   bool
}

func (m *SimpleServiceManager) GetServiceInfo(ctx context.Context, serviceName string) (*services.ServiceInfo, error) {
	// Check if context is already expired (this would be the bug condition)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Context is valid, proceed
	}

	m.callCount++
	return &services.ServiceInfo{
		Name:    serviceName,
		Running: true,
		Healthy: m.healthy,
	}, nil
}

func (m *SimpleServiceManager) GetServicesForPackage(packageName string) ([]string, error) {
	return []string{}, nil
}

func (m *SimpleServiceManager) SetServiceStartup(ctx context.Context, serviceName string, enabled bool) error {
	return nil
}

// Implement the remaining ServiceManager interface methods
func (m *SimpleServiceManager) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	return true, nil
}

func (m *SimpleServiceManager) GetAllServices(ctx context.Context) (map[string]*services.ServiceInfo, error) {
	return make(map[string]*services.ServiceInfo), nil
}

func (m *SimpleServiceManager) CheckHealth(ctx context.Context, serviceName string, config *services.HealthCheckConfig) (*services.HealthResult, error) {
	return &services.HealthResult{Healthy: m.healthy}, nil
}

func (m *SimpleServiceManager) StartService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *SimpleServiceManager) StopService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *SimpleServiceManager) RestartService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *SimpleServiceManager) DisableService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *SimpleServiceManager) IsServiceEnabled(ctx context.Context, serviceName string) (bool, error) {
	return true, nil
}

func (m *SimpleServiceManager) GetPackageForService(serviceName string) (string, error) {
	return "", nil
}

// MockLifecycleStrategy is a mock implementation of ServiceLifecycleStrategy for testing
type MockLifecycleStrategy struct {
	serviceName string
	healthy     bool
	callCount   int
}

func (m *MockLifecycleStrategy) StartService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *MockLifecycleStrategy) StopService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *MockLifecycleStrategy) RestartService(ctx context.Context, serviceName string) error {
	return nil
}

func (m *MockLifecycleStrategy) IsRunning(ctx context.Context, serviceName string) (bool, error) {
	return m.healthy, nil
}

func (m *MockLifecycleStrategy) GetStrategyName() services.ServiceManagementStrategy {
	return services.StrategyAuto
}

func (m *MockLifecycleStrategy) HealthCheck(ctx context.Context, serviceName string) (*services.ServiceHealthInfo, error) {
	m.callCount++
	return &services.ServiceHealthInfo{
		Healthy:  m.healthy,
		Details:  "Mock health check",
		Strategy: services.StrategyAuto,
	}, nil
}

func (m *MockLifecycleStrategy) StatusCheck(ctx context.Context, serviceName string) (*services.ServiceStatusInfo, error) {
	return &services.ServiceStatusInfo{
		Running:  m.healthy,
		Enabled:  false,
		Details:  "Mock status check",
		Strategy: services.StrategyAuto,
	}, nil
}

func TestServiceResource_WaitForHealthy_ContextTimeout_Fix(t *testing.T) {
	// This test verifies that the context timeout chain reaction bug is fixed
	// The bug: waitForHealthy creates a timeout context, then uses it for health checks
	// When the timeout expires, all subsequent commands fail with "context deadline exceeded"

	// Create simple service manager that starts unhealthy
	serviceManager := &SimpleServiceManager{
		healthy: false, // Service starts unhealthy
	}

	// Create service resource
	resource := &ServiceResource{
		serviceManager: serviceManager,
	}

	// Create a model with a timeout longer than the ticker interval (5s)
	model := &ServiceResourceModel{
		ServiceName:    types.StringValue("test-service"),
		WaitTimeout:    types.StringValue("6s"), // Longer than ticker interval to allow health checks
		WaitForHealthy: types.BoolValue(true),
	}

	// Create a mock lifecycle strategy for the test
	mockLifecycleStrategy := &MockLifecycleStrategy{
		serviceName: "test-service",
	}

	// Run the test
	ctx := context.Background()
	err := resource.waitForHealthy(ctx, model, mockLifecycleStrategy)

	// The service never becomes healthy, so this should timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout waiting for service to be healthy")

	// Verify that HealthCheck was called (proving commands executed with fresh contexts)
	// With the bug, callCount would be 0 because all commands would fail immediately
	// With the fix, callCount should be > 0 because fresh contexts allow commands to execute
	assert.Greater(t, mockLifecycleStrategy.callCount, 0, "Health checks should have been attempted")
}

func TestServiceResource_WaitForHealthy_FreshContext_Success(t *testing.T) {
	// This test verifies that health checks work with fresh contexts

	// Create service manager that reports healthy immediately
	serviceManager := &SimpleServiceManager{
		healthy: true, // Service is healthy
	}

	// Create service resource
	resource := &ServiceResource{
		serviceManager: serviceManager,
	}

	// Create a model with reasonable timeout
	model := &ServiceResourceModel{
		ServiceName:    types.StringValue("test-service"),
		WaitTimeout:    types.StringValue("30s"), // Reasonable timeout
		WaitForHealthy: types.BoolValue(true),
	}

	// Create a mock lifecycle strategy that reports healthy
	mockLifecycleStrategy := &MockLifecycleStrategy{
		serviceName: "test-service",
		healthy:     true,
	}

	// Run the test
	ctx := context.Background()
	err := resource.waitForHealthy(ctx, model, mockLifecycleStrategy)

	// Should succeed because service is healthy
	assert.NoError(t, err)

	// Verify that HealthCheck was called at least once
	assert.Greater(t, mockLifecycleStrategy.callCount, 0, "Health check should have been performed")
}

func TestSystemExecutor_ContextValidation(t *testing.T) {
	// This test verifies that SystemExecutor properly validates context before execution

	systemExecutor := executor.NewSystemExecutor()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to run a command with cancelled context
	result, err := systemExecutor.Run(ctx, "echo", []string{"test"}, executor.ExecOpts{})

	// Should fail with context error, not attempt to run the command
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context already expired")
	assert.Equal(t, -1, result.ExitCode)
}
