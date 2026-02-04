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

package files

import (
	"strings"
	"testing"
)

func TestUpdateYAMLFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		config         VersionFileConfig
		currentVersion string
		newVersion     string
		wantContain    string
		wantErr        bool
	}{
		{
			name: "simple path",
			input: `apiVersion: v1
metadata:
  name: test
  version: 1.0.0
`,
			config:         VersionFileConfig{Path: "metadata.version"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "version: 2.0.0",
		},
		{
			name: "nested path",
			input: `spec:
  template:
    spec:
      image:
        tag: v1.0.0
`,
			config:         VersionFileConfig{Path: "spec.template.spec.image.tag", Prefix: "v"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "tag: v2.0.0",
		},
		{
			name: "with prefix",
			input: `image:
  tag: v1.0.0
`,
			config:         VersionFileConfig{Path: "image.tag", Prefix: "v"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "tag: v2.0.0",
		},
		{
			name: "without prefix",
			input: `image:
  tag: 1.0.0
`,
			config:         VersionFileConfig{Path: "image.tag"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "tag: 2.0.0",
		},
		{
			name: "embedded version in image tag",
			input: `toolhiveRunnerImage: ghcr.io/stacklok/toolhive/proxyrunner:v0.7.1
`,
			config:         VersionFileConfig{Path: "toolhiveRunnerImage", Prefix: "v"},
			currentVersion: "0.7.1",
			newVersion:     "0.8.0",
			wantContain:    "toolhiveRunnerImage: ghcr.io/stacklok/toolhive/proxyrunner:v0.8.0",
		},
		{
			name: "embedded version without prefix",
			input: `image: myregistry.io/app:1.0.0-alpine
`,
			config:         VersionFileConfig{Path: "image"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "image: myregistry.io/app:2.0.0-alpine",
		},
		{
			name: "top level key",
			input: `version: 1.0.0
name: myapp
`,
			config:         VersionFileConfig{Path: "version"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContain:    "version: 2.0.0",
		},
		{
			name: "key not found",
			input: `metadata:
  name: test
`,
			config:         VersionFileConfig{Path: "metadata.version"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantErr:        true,
		},
		{
			name: "invalid path - missing parent",
			input: `metadata:
  name: test
`,
			config:         VersionFileConfig{Path: "spec.version"},
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "yaml-test-*.yaml")

			cfg := tt.config
			cfg.File = tmpPath

			err := UpdateYAMLFile(cfg, tt.currentVersion, tt.newVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateYAMLFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got := readTempFile(t, tmpPath)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("UpdateYAMLFile() result does not contain %q, got:\n%s", tt.wantContain, got)
			}
		})
	}
}

func TestUpdateYAMLFile_FileNotFound(t *testing.T) {
	t.Parallel()

	cfg := VersionFileConfig{
		File: "/nonexistent/path/file.yaml",
		Path: "version",
	}

	err := UpdateYAMLFile(cfg, "0.9.0", "1.0.0")
	if err == nil {
		t.Error("UpdateYAMLFile() expected error for nonexistent file")
	}
}

func TestUpdateYAMLFile_PreservesStructure(t *testing.T) {
	t.Parallel()

	input := `# This is a comment
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  labels:
    app: myapp
data:
  version: 1.0.0
  config: |
    some: yaml
    content: here
`

	tmpPath := createTempFile(t, input, "yaml-test-*.yaml")

	cfg := VersionFileConfig{
		File: tmpPath,
		Path: "data.version",
	}

	if err := UpdateYAMLFile(cfg, "1.0.0", "2.0.0"); err != nil {
		t.Fatalf("UpdateYAMLFile() error = %v", err)
	}

	content := readTempFile(t, tmpPath)

	// Check version was updated
	if !strings.Contains(content, "version: 2.0.0") {
		t.Errorf("version not updated, got:\n%s", content)
	}

	// Check comment preserved
	if !strings.Contains(content, "# This is a comment") {
		t.Errorf("comment lost, got:\n%s", content)
	}

	// Check other fields preserved
	if !strings.Contains(content, "kind: ConfigMap") {
		t.Errorf("kind field lost, got:\n%s", content)
	}
	if !strings.Contains(content, "app: myapp") {
		t.Errorf("labels lost, got:\n%s", content)
	}
}

func TestConvertToYAMLPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"version", "$.version", false},
		{"metadata.version", "$.metadata.version", false},
		{"spec.template.spec.image.tag", "$.spec.template.spec.image.tag", false},
		{"containers[0].image", "$.containers[0].image", false},
		{"$.already.prefixed", "$.already.prefixed", false},
		// Error cases
		{".image.tag", "", true},  // Leading dot
		{".version", "", true},    // Leading dot
		{"..recursive", "", true}, // Double dot
		{"", "", true},            // Empty path
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := convertToYAMLPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToYAMLPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("convertToYAMLPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractKeyFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"version", "version"},
		{"appVersion", "appVersion"},
		{"metadata.version", "version"},
		{"spec.template.spec.image.tag", "tag"},
		{"operator.image", "image"},
		{"containers[0].image", "image"},
		{"spec.containers[0].image", "image"},
		{"data[0]", "data"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := extractKeyFromPath(tt.input)
			if got != tt.want {
				t.Errorf("extractKeyFromPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUpdateYAMLFile_InvalidPath(t *testing.T) {
	t.Parallel()

	input := `image:
  tag: v1.0.0
`
	tmpPath := createTempFile(t, input, "yaml-test-*.yaml")

	cfg := VersionFileConfig{
		File: tmpPath,
		Path: ".image.tag",
	}

	err := UpdateYAMLFile(cfg, "1.0.0", "2.0.0")
	if err == nil {
		t.Error("UpdateYAMLFile() expected error for path starting with '.'")
	}
	if !strings.Contains(err.Error(), "cannot start with '.'") {
		t.Errorf("UpdateYAMLFile() error should mention leading dot, got: %v", err)
	}
}

func TestUpdateYAMLFile_PreservesQuotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantContain string
	}{
		{
			name: "preserves double quotes",
			input: `image:
  tag: "v1.0.0"
`,
			wantContain: `tag: "v2.0.0"`,
		},
		{
			name: "preserves single quotes",
			input: `image:
  tag: 'v1.0.0'
`,
			wantContain: `tag: 'v2.0.0'`,
		},
		{
			name: "preserves unquoted",
			input: `image:
  tag: v1.0.0
`,
			wantContain: `tag: v2.0.0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "yaml-test-*.yaml")

			cfg := VersionFileConfig{
				File:   tmpPath,
				Path:   "image.tag",
				Prefix: "v",
			}

			if err := UpdateYAMLFile(cfg, "1.0.0", "2.0.0"); err != nil {
				t.Fatalf("UpdateYAMLFile() error = %v", err)
			}

			got := readTempFile(t, tmpPath)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("UpdateYAMLFile() quote style not preserved, want %q in:\n%s", tt.wantContain, got)
			}
		})
	}
}

func TestUpdateYAMLFile_VersionMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		config         VersionFileConfig
		currentVersion string
		newVersion     string
		wantErrContain string
	}{
		{
			name: "image tag version mismatch",
			input: `operator:
  image: ghcr.io/stacklok/toolhive/operator:v0.8.1
`,
			config:         VersionFileConfig{Path: "operator.image", Prefix: "v"},
			currentVersion: "0.8.2",
			newVersion:     "0.8.3",
			wantErrContain: "version mismatch",
		},
		{
			name: "image tag version mismatch shows found version",
			input: `toolhiveRunnerImage: ghcr.io/stacklok/toolhive/proxyrunner:v0.7.1
`,
			config:         VersionFileConfig{Path: "toolhiveRunnerImage", Prefix: "v"},
			currentVersion: "0.8.0",
			newVersion:     "0.8.1",
			wantErrContain: "v0.7.1",
		},
		{
			name: "image tag version mismatch shows expected version",
			input: `image: registry.io/app:v1.0.0
`,
			config:         VersionFileConfig{Path: "image", Prefix: "v"},
			currentVersion: "2.0.0",
			newVersion:     "2.0.1",
			wantErrContain: "v2.0.0",
		},
		{
			name: "version mismatch without prefix",
			input: `image: myregistry.io/app:1.0.0
`,
			config:         VersionFileConfig{Path: "image"},
			currentVersion: "2.0.0",
			newVersion:     "2.0.1",
			wantErrContain: "version mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "yaml-test-*.yaml")

			cfg := tt.config
			cfg.File = tmpPath

			err := UpdateYAMLFile(cfg, tt.currentVersion, tt.newVersion)
			if err == nil {
				t.Error("UpdateYAMLFile() expected error for version mismatch")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("UpdateYAMLFile() error should contain %q, got: %v", tt.wantErrContain, err)
			}
		})
	}
}

func TestFindEmbeddedVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		prefix string
		want   string
	}{
		{
			name:   "image tag with v prefix",
			value:  "ghcr.io/stacklok/toolhive/operator:v0.8.1",
			prefix: "v",
			want:   "v0.8.1",
		},
		{
			name:   "image tag without prefix",
			value:  "myregistry.io/app:1.0.0",
			prefix: "",
			want:   "1.0.0",
		},
		{
			name:   "image tag with prerelease",
			value:  "registry.io/app:v1.0.0-alpha.1",
			prefix: "v",
			want:   "v1.0.0-alpha.1",
		},
		{
			name:   "simple version at end",
			value:  "v1.2.3",
			prefix: "v",
			want:   "v1.2.3",
		},
		{
			name:   "no version found",
			value:  "some-random-string",
			prefix: "v",
			want:   "",
		},
		{
			name:   "no version in plain text",
			value:  "hello world",
			prefix: "",
			want:   "",
		},
		{
			name:   "version in URL path (not tag)",
			value:  "https://example.com/v1/api",
			prefix: "v",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findEmbeddedVersion(tt.value, tt.prefix)
			if got != tt.want {
				t.Errorf("findEmbeddedVersion(%q, %q) = %q, want %q", tt.value, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestUpdateYAMLFile_PreservesComments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		path           string
		prefix         string
		currentVersion string
		newVersion     string
		wantContains   []string
	}{
		{
			name: "preserves inline comment after value",
			input: `image:
  tag: v1.0.0 # current version
`,
			path:           "image.tag",
			prefix:         "v",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"tag: v2.0.0 # current version",
			},
		},
		{
			name: "preserves comment above updated field",
			input: `image:
  # This is the image tag
  tag: v1.0.0
`,
			path:           "image.tag",
			prefix:         "v",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"# This is the image tag",
				"tag: v2.0.0",
			},
		},
		{
			name: "preserves comments between fields",
			input: `metadata:
  name: myapp
  # Version information
  version: 1.0.0
  # Author information
  author: test
`,
			path:           "metadata.version",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"# Version information",
				"version: 2.0.0",
				"# Author information",
				"author: test",
			},
		},
		{
			name: "preserves file header comment",
			input: `# This file is auto-generated
# Do not edit manually
apiVersion: v1
metadata:
  version: 1.0.0
`,
			path:           "metadata.version",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"# This file is auto-generated",
				"# Do not edit manually",
				"version: 2.0.0",
			},
		},
		{
			name: "preserves multiple inline comments",
			input: `spec:
  replicas: 3 # number of replicas
  image:
    tag: v1.0.0 # image version
    repo: myrepo # image repository
`,
			path:           "spec.image.tag",
			prefix:         "v",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"replicas: 3 # number of replicas",
				"tag: v2.0.0 # image version",
				"repo: myrepo # image repository",
			},
		},
		{
			name: "preserves block comments",
			input: `# ============================================
# Application Configuration
# ============================================
app:
  version: 1.0.0
  # Database settings
  database:
    host: localhost
`,
			path:           "app.version",
			currentVersion: "1.0.0",
			newVersion:     "2.0.0",
			wantContains: []string{
				"# ============================================",
				"# Application Configuration",
				"version: 2.0.0",
				"# Database settings",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "yaml-test-*.yaml")

			cfg := VersionFileConfig{
				File:   tmpPath,
				Path:   tt.path,
				Prefix: tt.prefix,
			}

			if err := UpdateYAMLFile(cfg, tt.currentVersion, tt.newVersion); err != nil {
				t.Fatalf("UpdateYAMLFile() error = %v", err)
			}

			got := readTempFile(t, tmpPath)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("UpdateYAMLFile() comment not preserved, want %q in:\n%s", want, got)
				}
			}
		})
	}
}

// TestUpdateYAMLFile_SameValueDifferentKeys tests that updating one field does not
// accidentally modify another field with the same value but different formatting.
// This was a bug where updating "version: 0.9.0" would also modify "appVersion: "0.9.0""
// because the replacement was not key-aware.
func TestUpdateYAMLFile_SameValueDifferentKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		config         VersionFileConfig
		currentVersion string
		newVersion     string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "update unquoted version without affecting quoted appVersion",
			input: `apiVersion: v2
name: test-chart
version: 0.9.0
appVersion: "0.9.0"
`,
			config:         VersionFileConfig{Path: "version"},
			currentVersion: "0.9.0",
			newVersion:     "0.9.1",
			wantContains: []string{
				"version: 0.9.1",
				`appVersion: "0.9.0"`, // appVersion should NOT be changed
			},
			wantNotContain: []string{
				`appVersion: "0.9.1"`, // should NOT happen
			},
		},
		{
			name: "update quoted appVersion without affecting unquoted version",
			input: `apiVersion: v2
name: test-chart
version: 0.9.0
appVersion: "0.9.0"
`,
			config:         VersionFileConfig{Path: "appVersion"},
			currentVersion: "0.9.0",
			newVersion:     "0.9.1",
			wantContains: []string{
				"version: 0.9.0", // version should NOT be changed
				`appVersion: "0.9.1"`,
			},
			wantNotContain: []string{
				"version: 0.9.1", // should NOT happen
			},
		},
		{
			name: "sequential updates to different keys with same value",
			input: `apiVersion: v2
name: operator-crds
version: 0.9.0
appVersion: "0.9.0"
`,
			config:         VersionFileConfig{Path: "version"},
			currentVersion: "0.9.0",
			newVersion:     "0.9.1",
			wantContains: []string{
				"version: 0.9.1",
				`appVersion: "0.9.0"`,
			},
		},
		{
			name: "different keys same value - nested paths",
			input: `chart:
  version: 0.9.0
  appVersion: "0.9.0"
  description: A test chart
`,
			config:         VersionFileConfig{Path: "chart.version"},
			currentVersion: "0.9.0",
			newVersion:     "0.9.1",
			wantContains: []string{
				"version: 0.9.1",
				`appVersion: "0.9.0"`, // appVersion should NOT be changed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "test*.yaml")
			cfg := tt.config
			cfg.File = tmpPath

			err := UpdateYAMLFile(cfg, tt.currentVersion, tt.newVersion)
			if err != nil {
				t.Fatalf("UpdateYAMLFile() error = %v", err)
			}

			got := readTempFile(t, tmpPath)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("UpdateYAMLFile() should contain %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("UpdateYAMLFile() should NOT contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}
