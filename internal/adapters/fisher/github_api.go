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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jamesainslie/terraform-provider-package/internal/executor"
)

// resolveLatestVersion resolves "latest" version to actual GitHub release version
func (f *FisherAdapter) resolveLatestVersion(ctx context.Context, pluginRef *PluginRef) (string, error) {
	// If not "latest", return the version as-is
	if pluginRef.Version != "latest" {
		return pluginRef.Version, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s/%s", pluginRef.Owner, pluginRef.Repo)
	if cached, exists := f.latestVersionCache[cacheKey]; exists {
		tflog.Debug(ctx, "Using cached version", map[string]interface{}{
			"plugin":  cacheKey,
			"version": cached,
		})
		return cached, nil
	}

	// Fetch latest release from GitHub API
	latest, err := f.fetchLatestGitHubRelease(ctx, pluginRef.Owner, pluginRef.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to resolve latest version for %s/%s: %w",
			pluginRef.Owner, pluginRef.Repo, err)
	}

	// Cache the result
	f.latestVersionCache[cacheKey] = latest

	tflog.Debug(ctx, "Resolved latest version", map[string]interface{}{
		"plugin":  cacheKey,
		"version": latest,
	})

	return latest, nil
}

// fetchLatestGitHubRelease fetches the latest release tag from GitHub API
func (f *FisherAdapter) fetchLatestGitHubRelease(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set User-Agent to avoid rate limiting
	req.Header.Set("User-Agent", "terraform-provider-package")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			tflog.Debug(ctx, "Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no release tag found")
	}

	return release.TagName, nil
}

// validateFishVersion checks if Fish shell version is compatible with Fisher
func (f *FisherAdapter) validateFishVersion(ctx context.Context) error {
	result, err := f.executor.Run(ctx, f.fishPath, []string{"--version"}, executor.ExecOpts{})
	if err != nil {
		return fmt.Errorf("failed to check Fish shell version: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("fish shell version check failed with exit code %d", result.ExitCode)
	}

	// Parse version from output like "fish, version 3.6.0"
	version, err := f.parseFishVersion(result.Stdout)
	if err != nil {
		return fmt.Errorf("unable to parse Fish shell version from output '%s': %w", result.Stdout, err)
	}

	// Check minimum version requirement (3.4.0)
	if !f.isValidFishVersion(version) {
		return fmt.Errorf("fisher requires Fish shell version 3.4.0 or later, found %s", version)
	}

	tflog.Debug(ctx, "Fish shell version validated", map[string]interface{}{
		"version": version,
		"output":  result.Stdout,
	})

	return nil
}

// parseFishVersion extracts version number from fish --version output
func (f *FisherAdapter) parseFishVersion(output string) (string, error) {
	// Expected format: "fish, version 3.6.0" or "fish, version 3.6.0-dirty"
	re := regexp.MustCompile(`fish,\s*version\s+(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(output)

	if len(matches) < 2 {
		return "", fmt.Errorf("version pattern not found in output")
	}

	return matches[1], nil
}

// isValidFishVersion checks if the version meets minimum requirements
func (f *FisherAdapter) isValidFishVersion(version string) bool {
	// Parse version numbers
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return false
	}

	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])

	if err1 != nil || err2 != nil || err3 != nil {
		return false
	}

	// Minimum version: 3.4.0
	if major > 3 {
		return true
	}
	if major == 3 {
		if minor > 4 {
			return true
		}
		if minor == 4 && patch >= 0 {
			return true
		}
	}

	return false
}
