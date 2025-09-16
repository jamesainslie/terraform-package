// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
	detail := fmt.Sprintf("Package '%s' was not found in %s. It may have been removed outside of Terraform.", packageName, manager)
	return diag.NewWarningDiagnostic(summary, detail)
}
