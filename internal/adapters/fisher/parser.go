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

package fisher

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// PluginRef represents a parsed Fisher plugin reference.
type PluginRef struct {
	Owner   string // GitHub owner/organization (empty for local plugins)
	Repo    string // GitHub repository name (empty for local plugins)
	Version string // Version/tag/branch (empty if not specified)
	IsLocal bool   // True if this is a local path plugin
	Path    string // Local filesystem path (empty for GitHub plugins)
	Raw     string // Original input string
}

// String returns the canonical string representation of the plugin reference.
func (p *PluginRef) String() string {
	if p.IsLocal {
		return p.Path
	}

	result := fmt.Sprintf("%s/%s", p.Owner, p.Repo)
	if p.Version != "" {
		result += "@" + p.Version
	}
	return result
}

// GitHubURL returns the GitHub repository URL for remote plugins.
func (p *PluginRef) GitHubURL() string {
	if p.IsLocal {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s", p.Owner, p.Repo)
}

// ParsePluginName parses various Fisher plugin name formats.
//
// Supported formats:
//   - GitHub repositories: "owner/repo", "owner/repo@version"
//   - Local paths: "/absolute/path", "~/relative/path", "./relative/path"
//
// Examples:
//   - "ilancosman/tide@v6" -> GitHub plugin with version
//   - "jorgebucaran/fisher" -> GitHub plugin without version
//   - "/path/to/local/plugin" -> Absolute local path
//   - "~/dev/my-plugin" -> Home-relative local path
func ParsePluginName(name string) (*PluginRef, error) {
	if name == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}

	// Create basic plugin reference
	ref := &PluginRef{
		Raw: name,
	}

	// Check if it's a local path (starts with /, ~/, or ./)
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "~/") || strings.HasPrefix(name, "./") {
		ref.IsLocal = true
		ref.Path = expandPath(name)
		return ref, nil
	}

	// Check if it's a GitHub repository reference - use more permissive regex first
	// But still ensure there's exactly one slash before the @ (if present)
	githubPattern := regexp.MustCompile(`^([^/@]+)/([^/@]+)(?:@(.+))?$`)
	matches := githubPattern.FindStringSubmatch(name)

	if len(matches) == 0 {
		return nil, fmt.Errorf("invalid plugin name format: %s (expected 'owner/repo[@version]' or local path)", name)
	}

	ref.Owner = matches[1]
	ref.Repo = matches[2]
	if len(matches) > 3 && matches[3] != "" {
		ref.Version = matches[3]
	}
	ref.IsLocal = false

	// Validate GitHub username/organization name
	if err := validateGitHubName(ref.Owner); err != nil {
		return nil, fmt.Errorf("invalid GitHub owner name '%s': %w", ref.Owner, err)
	}

	// Validate GitHub repository name
	if err := validateGitHubRepoName(ref.Repo); err != nil {
		return nil, fmt.Errorf("invalid GitHub repository name '%s': %w", ref.Repo, err)
	}

	return ref, nil
}

// expandPath expands ~ and relative paths to absolute paths.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		// This is a simplified expansion - in a real implementation,
		// you might want to use os.UserHomeDir() for proper handling
		return filepath.Join("$HOME", path[2:])
	}

	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err == nil {
			return abs
		}
	}

	return path
}

// validateGitHubName validates a GitHub username or organization name.
func validateGitHubName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 39 {
		return fmt.Errorf("name too long (max 39 characters)")
	}

	// GitHub usernames can only contain alphanumeric characters or hyphens
	// Cannot have multiple consecutive hyphens
	// Cannot begin or end with a hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("name cannot start or end with a hyphen")
	}

	if strings.Contains(name, "--") {
		return fmt.Errorf("name cannot contain consecutive hyphens")
	}

	githubNamePattern := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !githubNamePattern.MatchString(name) {
		return fmt.Errorf("name contains invalid characters (only alphanumeric and hyphens allowed)")
	}

	return nil
}

// validateGitHubRepoName validates a GitHub repository name.
func validateGitHubRepoName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("repository name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("repository name too long (max 100 characters)")
	}

	// Repository names can contain alphanumeric characters, hyphens, underscores, and periods
	// Cannot start with a period
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("repository name cannot start with a period")
	}

	repoNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	if !repoNamePattern.MatchString(name) {
		return fmt.Errorf("repository name contains invalid characters")
	}

	return nil
}
