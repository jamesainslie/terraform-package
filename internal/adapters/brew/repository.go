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

package brew

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jamesainslie/terraform-provider-package/internal/adapters"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

const (
	brewCommand = "brew"
)

// BrewRepositoryManager implements repository management for Homebrew taps.
type BrewRepositoryManager struct {
	executor executor.Executor
	brewPath string
}

// NewBrewRepositoryManager creates a new Homebrew repository manager.
func NewBrewRepositoryManager(exec executor.Executor, brewPath string) *BrewRepositoryManager {
	if brewPath == "" {
		brewPath = brewCommand
	}
	return &BrewRepositoryManager{
		executor: exec,
		brewPath: brewPath,
	}
}

// AddRepository adds a new Homebrew tap.
func (b *BrewRepositoryManager) AddRepository(ctx context.Context, name, uri, gpgKey string) error {
	// For Homebrew, the URI is the tap name (e.g., "homebrew/cask-fonts")
	// GPG keys are not used in Homebrew taps
	if gpgKey != "" {
		return fmt.Errorf("homebrew taps do not support GPG keys")
	}

	// Use the URI as the tap name
	tapName := uri
	if tapName == "" {
		tapName = name
	}

	result, err := b.executor.Run(ctx, b.brewPath, []string{"tap", tapName}, executor.ExecOpts{
		Timeout: 120 * time.Second, // 2 minutes for tap addition
	})

	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to add tap %s: exit code %d, error: %w, stderr: %s",
			tapName, result.ExitCode, err, result.Stderr)
	}

	return nil
}

// RemoveRepository removes a Homebrew tap.
func (b *BrewRepositoryManager) RemoveRepository(ctx context.Context, name string) error {
	// First try without force
	result, err := b.executor.Run(ctx, b.brewPath, []string{"untap", name}, executor.ExecOpts{
		Timeout: 60 * time.Second, // 1 minute for tap removal
	})

	// If it fails due to installed packages, try with --force
	if err != nil || result.ExitCode != 0 {
		if strings.Contains(result.Stderr, "contains the following installed") {
			// Try with force flag to remove tap even with installed packages
			result, err = b.executor.Run(ctx, b.brewPath, []string{"untap", "--force", name}, executor.ExecOpts{
				Timeout: 60 * time.Second,
			})
		}

		if err != nil || result.ExitCode != 0 {
			return fmt.Errorf("failed to remove tap %s: exit code %d, error: %w, stderr: %s",
				name, result.ExitCode, err, result.Stderr)
		}
	}

	return nil
}

// ListRepositories lists all configured Homebrew taps.
func (b *BrewRepositoryManager) ListRepositories(ctx context.Context) ([]adapters.RepositoryInfo, error) {
	result, err := b.executor.Run(ctx, b.brewPath, []string{"tap"}, executor.ExecOpts{
		Timeout: 30 * time.Second,
	})

	if err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list taps: exit code %d, error: %w, stderr: %s",
			result.ExitCode, err, result.Stderr)
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	repositories := make([]adapters.RepositoryInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		repositories = append(repositories, adapters.RepositoryInfo{
			Name:    line,
			URI:     line, // For Homebrew, name and URI are the same
			GPGKey:  "",   // Homebrew doesn't use GPG keys
			Enabled: true, // All listed taps are enabled
		})
	}

	return repositories, nil
}

// IsTapInstalled checks if a specific tap is installed.
func (b *BrewRepositoryManager) IsTapInstalled(ctx context.Context, tapName string) (bool, error) {
	repositories, err := b.ListRepositories(ctx)
	if err != nil {
		return false, err
	}

	for _, repo := range repositories {
		if repo.Name == tapName {
			return true, nil
		}
	}

	return false, nil
}
