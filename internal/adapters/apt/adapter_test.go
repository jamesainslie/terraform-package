package apt

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockExecutor for testing
type MockExecutor struct {
	mock.Mock
}

func (m *MockExecutor) Run(ctx context.Context, command string, args []string, opts executor.ExecOpts) (executor.ExecResult, error) {
	ret := m.Called(ctx, command, args, opts)
	return ret.Get(0).(executor.ExecResult), ret.Error(1)
}

func TestNewAptAdapter(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "/usr/bin/apt-get", "/usr/bin/dpkg-query", "/usr/bin/apt-cache")
	assert.NotNil(t, adapter)
	assert.Equal(t, "/usr/bin/apt-get", adapter.aptGetPath)
	assert.Equal(t, "/usr/bin/dpkg-query", adapter.dpkgPath)
	assert.Equal(t, "/usr/bin/apt-cache", adapter.aptCachePath)
}

func TestAptAdapter_GetManagerName(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")
	assert.Equal(t, "apt", adapter.GetManagerName())
}

func TestAptAdapter_IsAvailable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("APT adapter only available on Linux")
	}

	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Test successful case
	exec.On("Run", mock.Anything, "apt-get", []string{"--version"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: "apt 2.0"}, nil).
		Once()

	available := adapter.IsAvailable(context.Background())
	assert.True(t, available)

	exec.AssertExpectations(t)
}

func TestAptAdapter_DetectInstalled_Simple(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Test not installed (simple case)
	exec.On("Run", mock.Anything, "dpkg-query", mock.AnythingOfType("[]string"), mock.Anything).
		Return(executor.ExecResult{ExitCode: 1, Stderr: "no package"}, fmt.Errorf("not found")).
		Once()

	info, err := adapter.DetectInstalled(context.Background(), "notinstalled")
	assert.NoError(t, err)
	assert.False(t, info.Installed)
	assert.Equal(t, "notinstalled", info.Name)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Install_UpdateCache(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// With our idempotency enhancement, InstallWithType first calls DetectInstalled
	// Mock DetectInstalled to return not installed (so installation proceeds)
	exec.On("Run", mock.Anything, "dpkg-query",
		[]string{"--showformat", "${Package} ${Version} ${Status} ${Maintainer}", "--show", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 1, Stderr: "not found"}, fmt.Errorf("not found")).
		Once()

	// Test that install calls update first
	exec.On("Run", mock.Anything, "apt-get", []string{"update"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()
	exec.On("Run", mock.Anything, "apt-get", []string{"install", "-y", "--no-install-recommends", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err := adapter.InstallWithType(context.Background(), "testpkg", "", adapters.PackageTypeAuto)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Install_WithVersion(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// With our idempotency enhancement, InstallWithType first calls DetectInstalled
	// Mock DetectInstalled to return not installed (so installation proceeds)
	exec.On("Run", mock.Anything, "dpkg-query",
		[]string{"--showformat", "${Package} ${Version} ${Status} ${Maintainer}", "--show", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 1, Stderr: "not found"}, fmt.Errorf("not found")).
		Once()

	// Test install with version
	exec.On("Run", mock.Anything, "apt-get", []string{"update"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()
	exec.On("Run", mock.Anything, "apt-get", []string{"install", "-y", "--no-install-recommends", "testpkg=1.0"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err := adapter.InstallWithType(context.Background(), "testpkg", "1.0", adapters.PackageTypeAuto)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Remove(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Test remove
	exec.On("Run", mock.Anything, "apt-get", []string{"remove", "-y", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err := adapter.RemoveWithType(context.Background(), "testpkg", adapters.PackageTypeAuto)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Remove_NotInstalled_Idempotent(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Test idempotent remove (package not installed)
	exec.On("Run", mock.Anything, "apt-get", []string{"remove", "-y", "notpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 100, Stderr: "Package 'notpkg' is not installed, so not removed"}, nil).
		Once()

	err := adapter.RemoveWithType(context.Background(), "notpkg", adapters.PackageTypeAuto)
	assert.NoError(t, err) // Should be no-op, no error

	exec.AssertExpectations(t)
}

func TestAptAdapter_Pin(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Test pin (hold)
	exec.On("Run", mock.Anything, "apt-get", []string{"hold", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err := adapter.Pin(context.Background(), "testpkg", true)
	assert.NoError(t, err)

	// Test unpin (unhold)
	exec.On("Run", mock.Anything, "apt-get", []string{"unhold", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err = adapter.Pin(context.Background(), "testpkg", false)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_UpdateCache(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	exec.On("Run", mock.Anything, "apt-get", []string{"update"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	err := adapter.UpdateCache(context.Background())
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Search(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	searchOutput := `testpkg/testing - Test package
anotherpkg - Another package
`
	exec.On("Run", mock.Anything, "apt-get", []string{"search", "--no-install-recommends", "test"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: searchOutput}, nil).
		Once()

	results, err := adapter.Search(context.Background(), "test")
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "testpkg/testing", results[0].Name)
	assert.Equal(t, "anotherpkg", results[1].Name)
	assert.False(t, results[0].Installed) // Search doesn't set installed status

	exec.AssertExpectations(t)
}

func TestAptAdapter_Info_DelegatesToDetectInstalled(t *testing.T) {
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Info should delegate to DetectInstalled
	exec.On("Run", mock.Anything, "dpkg-query", mock.AnythingOfType("[]string"), mock.Anything).
		Return(executor.ExecResult{ExitCode: 1}, fmt.Errorf("not found")).
		Once()

	info, err := adapter.Info(context.Background(), "testpkg")
	assert.NoError(t, err)
	assert.False(t, info.Installed)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Install_Idempotency_AlreadyInstalled(t *testing.T) {
	// Test that InstallWithType skips installation if package is already installed with correct version
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Mock DetectInstalled calls: dpkg-query to return already installed with correct version
	exec.On("Run", mock.Anything, "dpkg-query",
		[]string{"--showformat", "${Package} ${Version} ${Status} ${Maintainer}", "--show", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: "testpkg 1.0 install ok installed"}, nil).
		Once()

	// Mock DetectInstalled calls: apt-cache policy for available versions
	exec.On("Run", mock.Anything, "apt-cache", []string{"policy", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: "Candidate: 1.0\nVersion table: *** 1.0"}, nil).
		Once()

	// UpdateCache and Install should NOT be called since package is already installed
	exec.AssertNotCalled(t, "Run", mock.Anything, "apt-get", []string{"update"}, mock.Anything)
	exec.AssertNotCalled(t, "Run", mock.Anything, "apt-get", []string{"install", "-y", "--no-install-recommends", "testpkg"}, mock.Anything)

	// This should return without error and without calling install
	err := adapter.InstallWithType(context.Background(), "testpkg", "", adapters.PackageTypeAuto)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}

func TestAptAdapter_Install_Idempotency_VersionMismatch(t *testing.T) {
	// Test that InstallWithType proceeds with installation if package is installed with wrong version
	exec := &MockExecutor{}
	adapter := NewAptAdapter(exec, "apt-get", "dpkg-query", "apt-cache")

	// Mock DetectInstalled calls: dpkg-query to return installed with different version
	exec.On("Run", mock.Anything, "dpkg-query",
		[]string{"--showformat", "${Package} ${Version} ${Status} ${Maintainer}", "--show", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: "testpkg 1.0 install ok installed"}, nil).
		Once()

	// Mock DetectInstalled calls: apt-cache policy for available versions
	exec.On("Run", mock.Anything, "apt-cache", []string{"policy", "testpkg"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0, Stdout: "Candidate: 2.0\nVersion table: *** 2.0"}, nil).
		Once()

	// UpdateCache and Install SHOULD be called since version differs
	exec.On("Run", mock.Anything, "apt-get", []string{"update"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()
	exec.On("Run", mock.Anything, "apt-get", []string{"install", "-y", "--no-install-recommends", "testpkg=2.0"}, mock.Anything).
		Return(executor.ExecResult{ExitCode: 0}, nil).
		Once()

	// This should proceed with installation since version differs (1.0 installed, 2.0 requested)
	err := adapter.InstallWithType(context.Background(), "testpkg", "2.0", adapters.PackageTypeAuto)
	assert.NoError(t, err)

	exec.AssertExpectations(t)
}
