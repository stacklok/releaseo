// Copyright 2025 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"
	"testing"

	"github.com/stacklok/releaseo/internal/files"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				BumpType:  "minor",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: false,
		},
		{
			name: "missing bump type",
			cfg: Config{
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: true,
			errMsg:  "--bump-type is required",
		},
		{
			name: "invalid bump type",
			cfg: Config{
				BumpType:  "invalid",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: true,
			errMsg:  "invalid bump type",
		},
		{
			name: "missing token",
			cfg: Config{
				BumpType:  "patch",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: true,
			errMsg:  "--token or GITHUB_TOKEN is required",
		},
		{
			name: "missing repo owner",
			cfg: Config{
				BumpType: "patch",
				Token:    "ghp_test",
				RepoName: "releaseo",
			},
			wantErr: true,
			errMsg:  "GITHUB_REPOSITORY environment variable is required",
		},
		{
			name: "missing repo name",
			cfg: Config{
				BumpType:  "patch",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
			},
			wantErr: true,
			errMsg:  "GITHUB_REPOSITORY environment variable is required",
		},
		{
			name: "all bump types valid - major",
			cfg: Config{
				BumpType:  "major",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: false,
		},
		{
			name: "all bump types valid - patch",
			cfg: Config{
				BumpType:  "patch",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: false,
		},
		{
			name: "bump type case insensitive",
			cfg: Config{
				BumpType:  "MAJOR",
				Token:     "ghp_test",
				RepoOwner: "stacklok",
				RepoName:  "releaseo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateConfig(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateConfig() unexpected error = %v", err)
			}
		})
	}
}

func TestParseVersionFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		jsonStr string
		want    []files.VersionFileConfig
		wantLen int
	}{
		{
			name:    "empty string",
			jsonStr: "",
			want:    nil,
			wantLen: 0,
		},
		{
			name:    "single file",
			jsonStr: `[{"file":"Chart.yaml","path":"version"}]`,
			wantLen: 1,
		},
		{
			name:    "multiple files",
			jsonStr: `[{"file":"Chart.yaml","path":"version"},{"file":"values.yaml","path":"image.tag","prefix":"v"}]`,
			wantLen: 2,
		},
		{
			name:    "with prefix",
			jsonStr: `[{"file":"values.yaml","path":"image.tag","prefix":"v"}]`,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseVersionFiles(tt.jsonStr)

			if len(got) != tt.wantLen {
				t.Errorf("parseVersionFiles() returned %d items, want %d", len(got), tt.wantLen)
			}

			if tt.want != nil && got == nil {
				t.Errorf("parseVersionFiles() = nil, want non-nil")
			}
		})
	}
}

func TestValidateHelmDocsArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty args",
			args:    "",
			wantErr: false,
		},
		{
			name:    "valid chart-search-root",
			args:    "--chart-search-root=./charts",
			wantErr: false,
		},
		{
			name:    "valid multiple args",
			args:    "--chart-search-root=./charts --template-files=README.md.gotmpl",
			wantErr: false,
		},
		{
			name:    "valid short flags",
			args:    "-c ./charts -t README.md.gotmpl",
			wantErr: false,
		},
		{
			name:    "invalid flag",
			args:    "--execute-script=malicious.sh",
			wantErr: true,
			errMsg:  "not allowed",
		},
		{
			name:    "mixed valid and invalid",
			args:    "--chart-search-root=./charts --invalid-flag=value",
			wantErr: true,
			errMsg:  "not allowed",
		},
		{
			name:    "valid dry-run",
			args:    "--dry-run",
			wantErr: false,
		},
		{
			name:    "valid log-level",
			args:    "--log-level=debug",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateHelmDocsArgs(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateHelmDocsArgs() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateHelmDocsArgs() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("validateHelmDocsArgs() unexpected error = %v", err)
			}
		})
	}
}

func TestGeneratePRBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		version      string
		bumpType     string
		versionFiles []files.VersionFileConfig
		ranHelmDocs  bool
		wantContains []string
	}{
		{
			name:         "basic PR body",
			version:      "1.2.3",
			bumpType:     "minor",
			versionFiles: nil,
			ranHelmDocs:  false,
			wantContains: []string{
				"Release v1.2.3",
				"**minor** release",
				"`VERSION`",
			},
		},
		{
			name:     "with version files",
			version:  "2.0.0",
			bumpType: "major",
			versionFiles: []files.VersionFileConfig{
				{File: "Chart.yaml", Path: "version"},
			},
			ranHelmDocs: false,
			wantContains: []string{
				"Release v2.0.0",
				"**major** release",
				"`Chart.yaml`",
				"path: `version`",
			},
		},
		{
			name:         "with helm docs",
			version:      "1.0.1",
			bumpType:     "patch",
			versionFiles: nil,
			ranHelmDocs:  true,
			wantContains: []string{
				"Release v1.0.1",
				"**patch** release",
				"helm-docs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generatePRBody(tt.version, tt.bumpType, tt.versionFiles, tt.ranHelmDocs)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("generatePRBody() missing %q in body:\n%s", want, got)
				}
			}
		})
	}
}

func TestGetModifiedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "only VERSION file",
			cfg: Config{
				VersionFile:  "VERSION",
				VersionFiles: nil,
			},
			want: []string{"VERSION"},
		},
		{
			name: "VERSION file with custom files",
			cfg: Config{
				VersionFile: "VERSION",
				VersionFiles: []files.VersionFileConfig{
					{File: "Chart.yaml", Path: "version"},
					{File: "values.yaml", Path: "image.tag"},
				},
			},
			want: []string{"VERSION", "Chart.yaml", "values.yaml"},
		},
		{
			name: "custom VERSION file location",
			cfg: Config{
				VersionFile:  "deploy/VERSION",
				VersionFiles: nil,
			},
			want: []string{"deploy/VERSION"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getModifiedFiles(tt.cfg)

			if len(got) != len(tt.want) {
				t.Errorf("getModifiedFiles() returned %d files, want %d", len(got), len(tt.want))
				return
			}

			for i, f := range got {
				if f != tt.want[i] {
					t.Errorf("getModifiedFiles()[%d] = %q, want %q", i, f, tt.want[i])
				}
			}
		})
	}
}

func TestResolveToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagToken string
		want      string
	}{
		{
			name:      "flag token provided",
			flagToken: "ghp_flagtoken",
			want:      "ghp_flagtoken",
		},
		{
			name:      "empty flag token returns empty (env not set in test)",
			flagToken: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Note: We don't set GITHUB_TOKEN env var in parallel tests
			// to avoid race conditions
			got := resolveToken(tt.flagToken)

			if got != tt.want {
				t.Errorf("resolveToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
