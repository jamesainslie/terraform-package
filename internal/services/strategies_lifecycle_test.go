package services

import (
	"context"
	"testing"

	"github.com/jamesainslie/terraform-provider-package/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExecutorForLifecycle is a mock executor for testing lifecycle strategies
type MockExecutorForLifecycle struct {
	mock.Mock
}

func (m *MockExecutorForLifecycle) Run(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	mockArgs := m.Called(ctx, command, args, opts)
	return mockArgs.Get(0).(executor.ExecResult), mockArgs.Error(1)
}

func TestDirectCommandLifecycleStrategy_HealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		customCommands  *CustomCommands
		mockSetup       func(*MockExecutorForLifecycle)
		expectedHealthy bool
		expectedDetails string
		expectError     bool
	}{
		{
			name:        "Colima running - healthy",
			serviceName: "colima",
			customCommands: &CustomCommands{
				Status: []string{"colima", "status"},
			},
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "colima", []string{"status"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 0,
						Stdout:   "colima is running",
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: true,
			expectedDetails: "Status command exit code: 0 (Colima VM is running)",
			expectError:     false,
		},
		{
			name:        "Colima not running - unhealthy",
			serviceName: "colima",
			customCommands: &CustomCommands{
				Status: []string{"colima", "status"},
			},
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "colima", []string{"status"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 1,
						Stdout:   "colima is not running",
						Stderr:   "VM not found",
					}, nil)
			},
			expectedHealthy: false,
			expectedDetails: "Status command exit code: 1, stderr: VM not found",
			expectError:     false,
		},
		{
			name:           "No status command - fallback to process check",
			serviceName:    "unknown-service",
			customCommands: &CustomCommands{}, // No status command
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "pgrep", []string{"-f", "unknown-service"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 0,
						Stdout:   "1234",
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: true,
			expectedDetails: "Process-based health check (no status command configured)",
			expectError:     false,
		},
		{
			name:           "No status command - process not found",
			serviceName:    "unknown-service",
			customCommands: &CustomCommands{}, // No status command
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "pgrep", []string{"-f", "unknown-service"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 1,
						Stdout:   "",
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: false,
			expectedDetails: "Process-based health check (no status command configured)",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutorForLifecycle{}
			tt.mockSetup(mockExecutor)

			strategy := &DirectCommandLifecycleStrategy{
				DirectCommandStrategy: DirectCommandStrategy{
					executor: mockExecutor,
					commands: tt.customCommands,
				},
			}

			ctx := context.Background()
			healthInfo, err := strategy.HealthCheck(ctx, tt.serviceName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHealthy, healthInfo.Healthy)
				assert.Contains(t, healthInfo.Details, tt.expectedDetails)
				assert.Equal(t, StrategyDirectCommand, healthInfo.Strategy)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestBrewServicesLifecycleStrategy_HealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		mockSetup       func(*MockExecutorForLifecycle)
		expectedHealthy bool
		expectedDetails string
		expectError     bool
	}{
		{
			name:        "Service running via brew services",
			serviceName: "postgresql",
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "brew", []string{"services", "list", "--json"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 0,
						Stdout:   `[{"name":"postgresql","status":"started","user":"test"}]`,
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: true,
			expectedDetails: "Service is started via brew services",
			expectError:     false,
		},
		{
			name:        "Service not running via brew services",
			serviceName: "postgresql",
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "brew", []string{"services", "list", "--json"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 0,
						Stdout:   `[{"name":"postgresql","status":"stopped","user":"test"}]`,
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: false,
			expectedDetails: "Service is not started via brew services",
			expectError:     false,
		},
		{
			name:        "Brew services command fails",
			serviceName: "postgresql",
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "brew", []string{"services", "list", "--json"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 1,
						Stdout:   "",
						Stderr:   "brew not found",
					}, nil)
			},
			expectedHealthy: false,
			expectedDetails: "brew services list failed: brew not found",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutorForLifecycle{}
			tt.mockSetup(mockExecutor)

			strategy := &BrewServicesLifecycleStrategy{
				BrewServicesStrategy: BrewServicesStrategy{
					executor: mockExecutor,
				},
			}

			ctx := context.Background()
			healthInfo, err := strategy.HealthCheck(ctx, tt.serviceName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHealthy, healthInfo.Healthy)
				assert.Contains(t, healthInfo.Details, tt.expectedDetails)
				assert.Equal(t, StrategyBrewServices, healthInfo.Strategy)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestProcessOnlyLifecycleStrategy_HealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		serviceName     string
		mockSetup       func(*MockExecutorForLifecycle)
		expectedHealthy bool
		expectedDetails string
		expectError     bool
	}{
		{
			name:        "Process found",
			serviceName: "test-service",
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "pgrep", []string{"-f", "test-service"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 0,
						Stdout:   "1234\n5678",
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: true,
			expectedDetails: "Process found (PIDs: 1234\n5678)",
			expectError:     false,
		},
		{
			name:        "Process not found",
			serviceName: "test-service",
			mockSetup: func(m *MockExecutorForLifecycle) {
				m.On("Run", mock.Anything, "pgrep", []string{"-f", "test-service"}, mock.Anything).
					Return(executor.ExecResult{
						ExitCode: 1,
						Stdout:   "",
						Stderr:   "",
					}, nil)
			},
			expectedHealthy: false,
			expectedDetails: "No matching process found",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutorForLifecycle{}
			tt.mockSetup(mockExecutor)

			strategy := &ProcessOnlyLifecycleStrategy{
				ProcessOnlyStrategy: ProcessOnlyStrategy{
					executor: mockExecutor,
				},
			}

			ctx := context.Background()
			healthInfo, err := strategy.HealthCheck(ctx, tt.serviceName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHealthy, healthInfo.Healthy)
				assert.Contains(t, healthInfo.Details, tt.expectedDetails)
				assert.Equal(t, StrategyProcessOnly, healthInfo.Strategy)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestServiceStrategyFactory_CreateLifecycleStrategy(t *testing.T) {
	mockExecutor := &MockExecutorForLifecycle{}
	factory := NewServiceStrategyFactory(mockExecutor)

	tests := []struct {
		name           string
		strategy       ServiceManagementStrategy
		customCommands *CustomCommands
		serviceName    string
		expectedType   string
	}{
		{
			name:           "Direct command strategy for Colima",
			strategy:       StrategyDirectCommand,
			customCommands: nil, // Should use defaults
			serviceName:    "colima",
			expectedType:   "*services.DirectCommandLifecycleStrategy",
		},
		{
			name:           "Brew services strategy",
			strategy:       StrategyBrewServices,
			customCommands: nil,
			serviceName:    "postgresql",
			expectedType:   "*services.BrewServicesLifecycleStrategy",
		},
		{
			name:           "Auto strategy should resolve to direct command for Colima",
			strategy:       StrategyAuto,
			customCommands: nil,
			serviceName:    "colima",
			expectedType:   "*services.DirectCommandLifecycleStrategy",
		},
		{
			name:           "Process only strategy",
			strategy:       StrategyProcessOnly,
			customCommands: nil,
			serviceName:    "test-service",
			expectedType:   "*services.ProcessOnlyLifecycleStrategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycleStrategy := factory.CreateLifecycleStrategy(tt.strategy, tt.customCommands, tt.serviceName)

			assert.NotNil(t, lifecycleStrategy)
			assert.Equal(t, tt.expectedType, getTypeName(lifecycleStrategy))

			// Verify it implements both interfaces
			assert.Implements(t, (*ServiceStrategy)(nil), lifecycleStrategy)
			assert.Implements(t, (*ServiceLifecycleStrategy)(nil), lifecycleStrategy)
		})
	}
}

// Helper function to get type name for testing
func getTypeName(obj interface{}) string {
	switch obj.(type) {
	case *DirectCommandLifecycleStrategy:
		return "*services.DirectCommandLifecycleStrategy"
	case *BrewServicesLifecycleStrategy:
		return "*services.BrewServicesLifecycleStrategy"
	case *AutoLifecycleStrategy:
		return "*services.AutoLifecycleStrategy"
	case *ProcessOnlyLifecycleStrategy:
		return "*services.ProcessOnlyLifecycleStrategy"
	case *LaunchdLifecycleStrategy:
		return "*services.LaunchdLifecycleStrategy"
	default:
		return "unknown"
	}
}
