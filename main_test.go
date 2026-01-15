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
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stacklok/releaseo/internal/files"
	"github.com/stacklok/releaseo/internal/github"
)

// mockVersionReader implements files.VersionReader for testing.
type mockVersionReader struct {
	version string
	err     error
}

func (m *mockVersionReader) ReadVersion(_ string) (string, error) {
	return m.version, m.err
}

// mockVersionWriter implements files.VersionWriter for testing.
type mockVersionWriter struct {
	err error
}

func (m *mockVersionWriter) WriteVersion(_, _ string) error {
	return m.err
}

// mockYAMLUpdater implements files.YAMLUpdater for testing.
type mockYAMLUpdater struct {
	err error
}

func (m *mockYAMLUpdater) UpdateYAMLFile(_ files.VersionFileConfig, _, _ string) error {
	return m.err
}

// mockPRCreator implements github.PRCreator for testing.
type mockPRCreator struct {
	result *github.PRResult
	err    error
}

func (m *mockPRCreator) CreateReleasePR(_ context.Context, _ github.PRRequest) (*github.PRResult, error) {
	return m.result, m.err
}

// TestUpdateResult_HasErrors tests the HasErrors method of UpdateResult.
func TestUpdateResult_HasErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		errors []error
		want   bool
	}{
		{
			name:   "empty errors returns false",
			errors: nil,
			want:   false,
		},
		{
			name:   "empty slice returns false",
			errors: []error{},
			want:   false,
		},
		{
			name:   "with single error returns true",
			errors: []error{errors.New("test error")},
			want:   true,
		},
		{
			name:   "with multiple errors returns true",
			errors: []error{errors.New("error 1"), errors.New("error 2")},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &UpdateResult{Errors: tt.errors}
			if got := r.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUpdateResult_CombinedError tests the CombinedError method of UpdateResult.
func TestUpdateResult_CombinedError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		errors      []error
		wantNil     bool
		wantStrings []string // substrings that should appear in the combined error
	}{
		{
			name:    "nil errors returns nil",
			errors:  nil,
			wantNil: true,
		},
		{
			name:    "empty slice returns nil",
			errors:  []error{},
			wantNil: true,
		},
		{
			name:        "single error is returned",
			errors:      []error{errors.New("single error")},
			wantNil:     false,
			wantStrings: []string{"single error"},
		},
		{
			name:        "multiple errors are combined",
			errors:      []error{errors.New("error one"), errors.New("error two")},
			wantNil:     false,
			wantStrings: []string{"error one", "error two"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := &UpdateResult{Errors: tt.errors}
			got := r.CombinedError()

			if tt.wantNil {
				if got != nil {
					t.Errorf("CombinedError() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("CombinedError() = nil, want non-nil error")
			}

			errStr := got.Error()
			for _, want := range tt.wantStrings {
				if !strings.Contains(errStr, want) {
					t.Errorf("CombinedError() = %q, want to contain %q", errStr, want)
				}
			}
		})
	}
}

// TestBumpVersion tests the bumpVersion function with various scenarios.
func TestBumpVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            Config
		reader         *mockVersionReader
		wantCurrent    string
		wantNewVersion string
		wantErr        bool
		errContains    string
	}{
		{
			name: "successful patch bump",
			cfg:  Config{BumpType: "patch", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "1.2.3",
				err:     nil,
			},
			wantCurrent:    "1.2.3",
			wantNewVersion: "1.2.4",
			wantErr:        false,
		},
		{
			name: "successful minor bump",
			cfg:  Config{BumpType: "minor", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "1.2.3",
				err:     nil,
			},
			wantCurrent:    "1.2.3",
			wantNewVersion: "1.3.0",
			wantErr:        false,
		},
		{
			name: "successful major bump",
			cfg:  Config{BumpType: "major", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "1.2.3",
				err:     nil,
			},
			wantCurrent:    "1.2.3",
			wantNewVersion: "2.0.0",
			wantErr:        false,
		},
		{
			name: "error reading version file",
			cfg:  Config{BumpType: "patch", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "",
				err:     errors.New("file not found"),
			},
			wantErr:     true,
			errContains: "reading version",
		},
		{
			name: "error parsing invalid version format",
			cfg:  Config{BumpType: "patch", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "invalid-version",
				err:     nil,
			},
			wantErr:     true,
			errContains: "parsing version",
		},
		{
			name: "error with invalid bump type",
			cfg:  Config{BumpType: "invalid", VersionFile: "VERSION"},
			reader: &mockVersionReader{
				version: "1.2.3",
				err:     nil,
			},
			wantErr:     true,
			errContains: "bumping version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			current, newVersion, err := bumpVersion(tt.cfg, tt.reader)

			if tt.wantErr {
				if err == nil {
					t.Fatal("bumpVersion() error = nil, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("bumpVersion() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("bumpVersion() unexpected error: %v", err)
			}

			if current != tt.wantCurrent {
				t.Errorf("bumpVersion() current = %q, want %q", current, tt.wantCurrent)
			}

			if newVersion.String() != tt.wantNewVersion {
				t.Errorf("bumpVersion() newVersion = %q, want %q", newVersion.String(), tt.wantNewVersion)
			}
		})
	}
}

// TestUpdateAllFiles tests the updateAllFiles function.
func TestUpdateAllFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		cfg            Config
		deps           *Dependencies
		wantHasErrors  bool
		wantErrorCount int
	}{
		{
			name: "success with no version files",
			cfg: Config{
				VersionFile:  "VERSION",
				HelmDocsArgs: "", // no helm-docs
			},
			deps: &Dependencies{
				VersionWriter: &mockVersionWriter{err: nil},
				YAMLUpdater:   &mockYAMLUpdater{err: nil},
			},
			wantHasErrors:  false,
			wantErrorCount: 0,
		},
		{
			name: "success with version files",
			cfg: Config{
				VersionFile: "VERSION",
				VersionFiles: []files.VersionFileConfig{
					{File: "chart/Chart.yaml", Path: "version"},
				},
				HelmDocsArgs: "", // no helm-docs
			},
			deps: &Dependencies{
				VersionWriter: &mockVersionWriter{err: nil},
				YAMLUpdater:   &mockYAMLUpdater{err: nil},
			},
			wantHasErrors:  false,
			wantErrorCount: 0,
		},
		{
			name: "version writer error",
			cfg: Config{
				VersionFile:  "VERSION",
				HelmDocsArgs: "",
			},
			deps: &Dependencies{
				VersionWriter: &mockVersionWriter{err: errors.New("write failed")},
				YAMLUpdater:   &mockYAMLUpdater{err: nil},
			},
			wantHasErrors:  true,
			wantErrorCount: 1,
		},
		{
			name: "yaml updater error",
			cfg: Config{
				VersionFile: "VERSION",
				VersionFiles: []files.VersionFileConfig{
					{File: "chart/Chart.yaml", Path: "version"},
				},
				HelmDocsArgs: "",
			},
			deps: &Dependencies{
				VersionWriter: &mockVersionWriter{err: nil},
				YAMLUpdater:   &mockYAMLUpdater{err: errors.New("yaml update failed")},
			},
			wantHasErrors:  true,
			wantErrorCount: 1,
		},
		{
			name: "multiple errors collected",
			cfg: Config{
				VersionFile: "VERSION",
				VersionFiles: []files.VersionFileConfig{
					{File: "chart/Chart.yaml", Path: "version"},
				},
				HelmDocsArgs: "",
			},
			deps: &Dependencies{
				VersionWriter: &mockVersionWriter{err: errors.New("write failed")},
				YAMLUpdater:   &mockYAMLUpdater{err: errors.New("yaml update failed")},
			},
			wantHasErrors:  true,
			wantErrorCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := updateAllFiles(tt.cfg, "1.0.0", "1.0.1", tt.deps)

			if result.HasErrors() != tt.wantHasErrors {
				t.Errorf("updateAllFiles() HasErrors() = %v, want %v", result.HasErrors(), tt.wantHasErrors)
			}

			if len(result.Errors) != tt.wantErrorCount {
				t.Errorf("updateAllFiles() error count = %d, want %d", len(result.Errors), tt.wantErrorCount)
			}
		})
	}
}

// TestCreateReleasePR tests the createReleasePR function.
func TestCreateReleasePR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cfg           Config
		prCreator     *mockPRCreator
		newVersion    string
		helmDocsFiles []string
		wantErr       bool
		errContains   string
		wantPRNumber  int
		wantPRURL     string
	}{
		{
			name: "success",
			cfg: Config{
				RepoOwner:   "owner",
				RepoName:    "repo",
				BaseBranch:  "main",
				BumpType:    "patch",
				VersionFile: "VERSION",
			},
			prCreator: &mockPRCreator{
				result: &github.PRResult{
					Number: 123,
					URL:    "https://github.com/owner/repo/pull/123",
				},
				err: nil,
			},
			newVersion:   "1.0.1",
			wantErr:      false,
			wantPRNumber: 123,
			wantPRURL:    "https://github.com/owner/repo/pull/123",
		},
		{
			name: "success with helm docs files",
			cfg: Config{
				RepoOwner:    "owner",
				RepoName:     "repo",
				BaseBranch:   "main",
				BumpType:     "patch",
				VersionFile:  "VERSION",
				HelmDocsArgs: "-c charts/",
			},
			prCreator: &mockPRCreator{
				result: &github.PRResult{
					Number: 456,
					URL:    "https://github.com/owner/repo/pull/456",
				},
				err: nil,
			},
			newVersion:    "2.0.0",
			helmDocsFiles: []string{"charts/README.md"},
			wantErr:       false,
			wantPRNumber:  456,
			wantPRURL:     "https://github.com/owner/repo/pull/456",
		},
		{
			name: "error from pr creator",
			cfg: Config{
				RepoOwner:   "owner",
				RepoName:    "repo",
				BaseBranch:  "main",
				BumpType:    "patch",
				VersionFile: "VERSION",
			},
			prCreator: &mockPRCreator{
				result: nil,
				err:    errors.New("github api error"),
			},
			newVersion:  "1.0.1",
			wantErr:     true,
			errContains: "creating PR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			result, err := createReleasePR(ctx, tt.cfg, tt.prCreator, tt.newVersion, tt.helmDocsFiles)

			if tt.wantErr {
				if err == nil {
					t.Fatal("createReleasePR() error = nil, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("createReleasePR() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("createReleasePR() unexpected error: %v", err)
			}

			if result.Number != tt.wantPRNumber {
				t.Errorf("createReleasePR() PR number = %d, want %d", result.Number, tt.wantPRNumber)
			}

			if result.URL != tt.wantPRURL {
				t.Errorf("createReleasePR() PR URL = %q, want %q", result.URL, tt.wantPRURL)
			}
		})
	}
}

// TestGeneratePRBody tests the generatePRBody function.
func TestGeneratePRBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		version      string
		bumpType     string
		versionFiles []files.VersionFileConfig
		ranHelmDocs  bool
		wantStrings  []string
		dontWant     []string
	}{
		{
			name:         "basic case with no version files",
			version:      "1.0.0",
			bumpType:     "patch",
			versionFiles: nil,
			ranHelmDocs:  false,
			wantStrings: []string{
				"## Release v1.0.0",
				"**patch** release",
				"- `VERSION`",
				"### Next Steps",
				"### Checklist",
			},
			dontWant: []string{
				"helm-docs",
			},
		},
		{
			name:     "with version files",
			version:  "2.0.0",
			bumpType: "major",
			versionFiles: []files.VersionFileConfig{
				{File: "chart/Chart.yaml", Path: "version"},
				{File: "app/values.yaml", Path: "image.tag"},
			},
			ranHelmDocs: false,
			wantStrings: []string{
				"## Release v2.0.0",
				"**major** release",
				"- `VERSION`",
				"- `chart/Chart.yaml` (path: `version`)",
				"- `app/values.yaml` (path: `image.tag`)",
			},
			dontWant: []string{
				"helm-docs",
			},
		},
		{
			name:         "with helm-docs",
			version:      "1.5.0",
			bumpType:     "minor",
			versionFiles: nil,
			ranHelmDocs:  true,
			wantStrings: []string{
				"## Release v1.5.0",
				"**minor** release",
				"- `VERSION`",
				"Helm chart docs (via helm-docs)",
			},
		},
		{
			name:     "with version files and helm-docs",
			version:  "3.0.0",
			bumpType: "major",
			versionFiles: []files.VersionFileConfig{
				{File: "charts/app/Chart.yaml", Path: "appVersion"},
			},
			ranHelmDocs: true,
			wantStrings: []string{
				"## Release v3.0.0",
				"**major** release",
				"- `VERSION`",
				"- `charts/app/Chart.yaml` (path: `appVersion`)",
				"Helm chart docs (via helm-docs)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := generatePRBody(tt.version, tt.bumpType, tt.versionFiles, tt.ranHelmDocs)

			for _, want := range tt.wantStrings {
				if !strings.Contains(body, want) {
					t.Errorf("generatePRBody() = %q, want to contain %q", body, want)
				}
			}

			for _, dontWant := range tt.dontWant {
				if strings.Contains(body, dontWant) {
					t.Errorf("generatePRBody() = %q, should not contain %q", body, dontWant)
				}
			}
		})
	}
}

// TestGetModifiedFiles tests the getModifiedFiles function.
func TestGetModifiedFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       Config
		wantFiles []string
	}{
		{
			name: "version file only",
			cfg: Config{
				VersionFile: "VERSION",
			},
			wantFiles: []string{"VERSION"},
		},
		{
			name: "version file with custom files",
			cfg: Config{
				VersionFile: "VERSION",
				VersionFiles: []files.VersionFileConfig{
					{File: "chart/Chart.yaml", Path: "version"},
					{File: "app/values.yaml", Path: "image.tag"},
				},
			},
			wantFiles: []string{"VERSION", "chart/Chart.yaml", "app/values.yaml"},
		},
		{
			name: "custom version file path",
			cfg: Config{
				VersionFile: "config/VERSION.txt",
			},
			wantFiles: []string{"config/VERSION.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getModifiedFiles(tt.cfg)

			if len(got) != len(tt.wantFiles) {
				t.Errorf("getModifiedFiles() returned %d files, want %d", len(got), len(tt.wantFiles))
			}

			for i, want := range tt.wantFiles {
				if i >= len(got) {
					t.Errorf("getModifiedFiles() missing file at index %d: want %q", i, want)
					continue
				}
				if got[i] != want {
					t.Errorf("getModifiedFiles()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
