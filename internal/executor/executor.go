// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package executor

import (
	"context"
	"time"
)

// ExecResult represents the result of a command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecOpts provides options for command execution.
type ExecOpts struct {
	Timeout        time.Duration
	WorkDir        string
	Env            []string
	UseSudo        bool
	NonInteractive bool
}

// Executor defines the interface for executing system commands.
type Executor interface {
	Run(ctx context.Context, cmd string, args []string, opts ExecOpts) (ExecResult, error)
}
