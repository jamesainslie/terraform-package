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
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// PackageError represents a package management error.
type PackageError struct {
	Operation string
	Package   string
	Manager   string
	Cause     error
	ExitCode  int
}

func (e *PackageError) Error() string {
	return fmt.Sprintf("package %s failed for package %s using %s: %v (exit code: %d)",
		e.Operation, e.Package, e.Manager, e.Cause, e.ExitCode)
}

// NewPackageError creates a new PackageError.
func NewPackageError(operation, packageName, manager string, cause error, exitCode int) *PackageError {
	return &PackageError{
		Operation: operation,
		Package:   packageName,
		Manager:   manager,
		Cause:     cause,
		ExitCode:  exitCode,
	}
}

// PrivilegeError represents a privilege/permission error.
type PrivilegeError struct {
	Operation string
	Message   string
	Guidance  string
}

func (e *PrivilegeError) Error() string {
	if e.Guidance != "" {
		return fmt.Sprintf("%s operation failed: %s. Guidance: %s", e.Operation, e.Message, e.Guidance)
	}
	return fmt.Sprintf("%s operation failed: %s", e.Operation, e.Message)
}

// NewPrivilegeError creates a new PrivilegeError.
func NewPrivilegeError(operation, message, guidance string) *PrivilegeError {
	return &PrivilegeError{
		Operation: operation,
		Message:   message,
		Guidance:  guidance,
	}
}

// DiagnosticHelpers provides utilities for creating Terraform diagnostics.
type DiagnosticHelpers struct{}

// NewDiagnosticHelpers creates a new DiagnosticHelpers instance.
func NewDiagnosticHelpers() *DiagnosticHelpers {
	return &DiagnosticHelpers{}
}

// ErrorDiagnostic creates an error diagnostic from an error.
func (d *DiagnosticHelpers) ErrorDiagnostic(summary string, err error) diag.Diagnostic {
	return diag.NewErrorDiagnostic(summary, err.Error())
}

// WarningDiagnostic creates a warning diagnostic.
func (d *DiagnosticHelpers) WarningDiagnostic(summary, detail string) diag.Diagnostic {
	return diag.NewWarningDiagnostic(summary, detail)
}

// PackageErrorDiagnostic creates a diagnostic from a PackageError.
func (d *DiagnosticHelpers) PackageErrorDiagnostic(err *PackageError) diag.Diagnostic {
	summary := fmt.Sprintf("Package %s Failed", err.Operation)
	detail := fmt.Sprintf("Failed to %s package '%s' using %s manager. Exit code: %d. Error: %v",
		err.Operation, err.Package, err.Manager, err.ExitCode, err.Cause)
	return diag.NewErrorDiagnostic(summary, detail)
}

// PrivilegeErrorDiagnostic creates a diagnostic from a PrivilegeError.
func (d *DiagnosticHelpers) PrivilegeErrorDiagnostic(err *PrivilegeError) diag.Diagnostic {
	summary := "Insufficient Privileges"
	detail := fmt.Sprintf("Operation '%s' requires elevated privileges: %s", err.Operation, err.Message)
	if err.Guidance != "" {
		detail += fmt.Sprintf("\n\nGuidance: %s", err.Guidance)
	}
	return diag.NewErrorDiagnostic(summary, detail)
}

// ValidationErrorDiagnostic creates a validation error diagnostic.
func (d *DiagnosticHelpers) ValidationErrorDiagnostic(field, message string) diag.Diagnostic {
	summary := fmt.Sprintf("Invalid %s", field)
	return diag.NewErrorDiagnostic(summary, message)
}

// NotFoundWarningDiagnostic creates a warning for when a package is not found.
func (d *DiagnosticHelpers) NotFoundWarningDiagnostic(packageName, manager string) diag.Diagnostic {
	summary := "Package Not Found"
	detail := fmt.Sprintf("Package '%s' was not found in %s. It may have been removed outside of Terraform.",
		packageName, manager)
	return diag.NewWarningDiagnostic(summary, detail)
}
