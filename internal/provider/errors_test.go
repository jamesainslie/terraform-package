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
	"errors"
	"strings"
	"testing"
)

func TestPackageError(t *testing.T) {
	cause := errors.New("command failed")
	err := NewPackageError("install", "git", "brew", cause, 1)

	if err.Operation != "install" {
		t.Errorf("Expected operation 'install', got '%s'", err.Operation)
	}

	if err.Package != "git" {
		t.Errorf("Expected package 'git', got '%s'", err.Package)
	}

	if err.Manager != "brew" {
		t.Errorf("Expected manager 'brew', got '%s'", err.Manager)
	}

	if err.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", err.ExitCode)
	}

	if !errors.Is(err.Cause, cause) {
		t.Errorf("Expected cause to be the original error")
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "install") {
		t.Errorf("Error message should contain operation: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "git") {
		t.Errorf("Error message should contain package: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "brew") {
		t.Errorf("Error message should contain manager: %s", errorMsg)
	}
}

func TestPrivilegeError(t *testing.T) {
	err := NewPrivilegeError("install", "sudo required", "run as root")

	if err.Operation != "install" {
		t.Errorf("Expected operation 'install', got '%s'", err.Operation)
	}

	if err.Message != "sudo required" {
		t.Errorf("Expected message 'sudo required', got '%s'", err.Message)
	}

	if err.Guidance != "run as root" {
		t.Errorf("Expected guidance 'run as root', got '%s'", err.Guidance)
	}

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "install") {
		t.Errorf("Error message should contain operation: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "sudo required") {
		t.Errorf("Error message should contain message: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "run as root") {
		t.Errorf("Error message should contain guidance: %s", errorMsg)
	}
}

func TestPrivilegeError_NoGuidance(t *testing.T) {
	err := NewPrivilegeError("remove", "permission denied", "")

	errorMsg := err.Error()
	if !strings.Contains(errorMsg, "remove") {
		t.Errorf("Error message should contain operation: %s", errorMsg)
	}
	if !strings.Contains(errorMsg, "permission denied") {
		t.Errorf("Error message should contain message: %s", errorMsg)
	}

	// Should not contain "Guidance:" when guidance is empty
	if strings.Contains(errorMsg, "Guidance:") {
		t.Errorf("Error message should not contain guidance when empty: %s", errorMsg)
	}
}

func TestDiagnosticHelpers_ErrorDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()
	testErr := errors.New("test error")

	diag := helpers.ErrorDiagnostic("Test Summary", testErr)

	if diag.Summary() != "Test Summary" {
		t.Errorf("Expected summary 'Test Summary', got '%s'", diag.Summary())
	}

	if !strings.Contains(diag.Detail(), "test error") {
		t.Errorf("Expected detail to contain error message: %s", diag.Detail())
	}
}

func TestDiagnosticHelpers_WarningDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()

	diag := helpers.WarningDiagnostic("Warning Summary", "Warning detail")

	if diag.Summary() != "Warning Summary" {
		t.Errorf("Expected summary 'Warning Summary', got '%s'", diag.Summary())
	}

	if diag.Detail() != "Warning detail" {
		t.Errorf("Expected detail 'Warning detail', got '%s'", diag.Detail())
	}
}

func TestDiagnosticHelpers_PackageErrorDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()
	cause := errors.New("command failed")
	pkgErr := NewPackageError("install", "git", "brew", cause, 1)

	diag := helpers.PackageErrorDiagnostic(pkgErr)

	summary := diag.Summary()
	if !strings.Contains(summary, "Package") {
		t.Errorf("Expected summary to contain 'Package': %s", summary)
	}
	if !strings.Contains(summary, "install") {
		t.Errorf("Expected summary to contain operation: %s", summary)
	}

	detail := diag.Detail()
	if !strings.Contains(detail, "git") {
		t.Errorf("Expected detail to contain package name: %s", detail)
	}
	if !strings.Contains(detail, "brew") {
		t.Errorf("Expected detail to contain manager: %s", detail)
	}
	if !strings.Contains(detail, "1") {
		t.Errorf("Expected detail to contain exit code: %s", detail)
	}
}

func TestDiagnosticHelpers_PrivilegeErrorDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()
	privErr := NewPrivilegeError("install", "sudo required", "run as root")

	diag := helpers.PrivilegeErrorDiagnostic(privErr)

	summary := diag.Summary()
	if summary != "Insufficient Privileges" {
		t.Errorf("Expected summary 'Insufficient Privileges', got '%s'", summary)
	}

	detail := diag.Detail()
	if !strings.Contains(detail, "install") {
		t.Errorf("Expected detail to contain operation: %s", detail)
	}
	if !strings.Contains(detail, "sudo required") {
		t.Errorf("Expected detail to contain message: %s", detail)
	}
	if !strings.Contains(detail, "run as root") {
		t.Errorf("Expected detail to contain guidance: %s", detail)
	}
}

func TestDiagnosticHelpers_ValidationErrorDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()

	diag := helpers.ValidationErrorDiagnostic("package_name", "package name cannot be empty")

	summary := diag.Summary()
	if !strings.Contains(summary, "Invalid package_name") {
		t.Errorf("Expected summary to contain field name: %s", summary)
	}

	detail := diag.Detail()
	if detail != "package name cannot be empty" {
		t.Errorf("Expected detail 'package name cannot be empty', got '%s'", detail)
	}
}

func TestDiagnosticHelpers_NotFoundWarningDiagnostic(t *testing.T) {
	helpers := NewDiagnosticHelpers()

	diag := helpers.NotFoundWarningDiagnostic("git", "homebrew")

	summary := diag.Summary()
	if summary != "Package Not Found" {
		t.Errorf("Expected summary 'Package Not Found', got '%s'", summary)
	}

	detail := diag.Detail()
	if !strings.Contains(detail, "git") {
		t.Errorf("Expected detail to contain package name: %s", detail)
	}
	if !strings.Contains(detail, "homebrew") {
		t.Errorf("Expected detail to contain manager: %s", detail)
	}
	if !strings.Contains(detail, "outside of Terraform") {
		t.Errorf("Expected detail to contain drift explanation: %s", detail)
	}
}
