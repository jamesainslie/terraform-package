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
	"testing"
)

func TestParsePluginName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *PluginRef
		wantErr     bool
		errContains string
	}{
		{
			name:  "GitHub plugin without version",
			input: "ilancosman/tide",
			want: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				Version: "",
				IsLocal: false,
				Path:    "",
				Raw:     "ilancosman/tide",
			},
			wantErr: false,
		},
		{
			name:  "GitHub plugin with version",
			input: "ilancosman/tide@v6",
			want: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				Version: "v6",
				IsLocal: false,
				Path:    "",
				Raw:     "ilancosman/tide@v6",
			},
			wantErr: false,
		},
		{
			name:  "GitHub plugin with branch",
			input: "jorgebucaran/fisher@main",
			want: &PluginRef{
				Owner:   "jorgebucaran",
				Repo:    "fisher",
				Version: "main",
				IsLocal: false,
				Path:    "",
				Raw:     "jorgebucaran/fisher@main",
			},
			wantErr: false,
		},
		{
			name:  "absolute local path",
			input: "/Users/dev/my-plugin",
			want: &PluginRef{
				Owner:   "",
				Repo:    "",
				Version: "",
				IsLocal: true,
				Path:    "/Users/dev/my-plugin",
				Raw:     "/Users/dev/my-plugin",
			},
			wantErr: false,
		},
		{
			name:  "home-relative local path",
			input: "~/dev/my-plugin",
			want: &PluginRef{
				Owner:   "",
				Repo:    "",
				Version: "",
				IsLocal: true,
				Path:    "$HOME/dev/my-plugin",
				Raw:     "~/dev/my-plugin",
			},
			wantErr: false,
		},
		{
			name:  "current-relative local path",
			input: "./my-plugin",
			want: &PluginRef{
				Owner:   "",
				Repo:    "",
				Version: "",
				IsLocal: true,
				// Path will be absolute, so we'll check IsLocal instead
				Raw: "./my-plugin",
			},
			wantErr: false,
		},
		{
			name:        "empty plugin name",
			input:       "",
			want:        nil,
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "invalid GitHub format - no slash",
			input:       "invalid-plugin",
			want:        nil,
			wantErr:     true,
			errContains: "invalid plugin name format",
		},
		{
			name:        "invalid GitHub format - too many slashes",
			input:       "owner/repo/extra",
			want:        nil,
			wantErr:     true,
			errContains: "invalid plugin name format",
		},
		{
			name:        "invalid GitHub owner - starts with hyphen",
			input:       "-invalid/repo",
			want:        nil,
			wantErr:     true,
			errContains: "cannot start or end with a hyphen",
		},
		{
			name:        "invalid GitHub owner - ends with hyphen",
			input:       "invalid-/repo",
			want:        nil,
			wantErr:     true,
			errContains: "cannot start or end with a hyphen",
		},
		{
			name:        "invalid GitHub owner - consecutive hyphens",
			input:       "inva--lid/repo",
			want:        nil,
			wantErr:     true,
			errContains: "consecutive hyphens",
		},
		{
			name:        "invalid GitHub repo - starts with period",
			input:       "owner/.invalid",
			want:        nil,
			wantErr:     true,
			errContains: "cannot start with a period",
		},
		{
			name:  "valid repo with underscores and periods",
			input: "owner/valid_repo.name",
			want: &PluginRef{
				Owner:   "owner",
				Repo:    "valid_repo.name",
				Version: "",
				IsLocal: false,
				Path:    "",
				Raw:     "owner/valid_repo.name",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePluginName(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePluginName() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ParsePluginName() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePluginName() unexpected error = %v", err)
				return
			}

			if got == nil {
				t.Errorf("ParsePluginName() returned nil result")
				return
			}

			// Check all fields except Path for relative paths (which get expanded)
			if got.Owner != tt.want.Owner {
				t.Errorf("ParsePluginName() Owner = %v, want %v", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("ParsePluginName() Repo = %v, want %v", got.Repo, tt.want.Repo)
			}
			if got.Version != tt.want.Version {
				t.Errorf("ParsePluginName() Version = %v, want %v", got.Version, tt.want.Version)
			}
			if got.IsLocal != tt.want.IsLocal {
				t.Errorf("ParsePluginName() IsLocal = %v, want %v", got.IsLocal, tt.want.IsLocal)
			}
			if got.Raw != tt.want.Raw {
				t.Errorf("ParsePluginName() Raw = %v, want %v", got.Raw, tt.want.Raw)
			}

			// For non-relative local paths, check exact path
			if tt.want.Path != "" && got.Path != tt.want.Path {
				t.Errorf("ParsePluginName() Path = %v, want %v", got.Path, tt.want.Path)
			}

			// For relative paths, just ensure path is not empty
			if tt.input == "./my-plugin" && !got.IsLocal {
				t.Errorf("ParsePluginName() expected relative path to be marked as local")
			}
		})
	}
}

func TestPluginRef_String(t *testing.T) {
	tests := []struct {
		name string
		ref  *PluginRef
		want string
	}{
		{
			name: "GitHub plugin without version",
			ref: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				IsLocal: false,
			},
			want: "ilancosman/tide",
		},
		{
			name: "GitHub plugin with version",
			ref: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				Version: "v6",
				IsLocal: false,
			},
			want: "ilancosman/tide@v6",
		},
		{
			name: "local plugin",
			ref: &PluginRef{
				Path:    "/path/to/plugin",
				IsLocal: true,
			},
			want: "/path/to/plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("PluginRef.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluginRef_GitHubURL(t *testing.T) {
	tests := []struct {
		name string
		ref  *PluginRef
		want string
	}{
		{
			name: "GitHub plugin",
			ref: &PluginRef{
				Owner:   "ilancosman",
				Repo:    "tide",
				IsLocal: false,
			},
			want: "https://github.com/ilancosman/tide",
		},
		{
			name: "local plugin",
			ref: &PluginRef{
				Path:    "/path/to/plugin",
				IsLocal: true,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.GitHubURL()
			if got != tt.want {
				t.Errorf("PluginRef.GitHubURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

