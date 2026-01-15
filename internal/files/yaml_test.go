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
		name        string
		input       string
		config      VersionFileConfig
		version     string
		wantContain string
		wantErr     bool
	}{
		{
			name: "simple path",
			input: `apiVersion: v1
metadata:
  name: test
  version: 1.0.0
`,
			config:      VersionFileConfig{Path: "metadata.version"},
			version:     "2.0.0",
			wantContain: "version: 2.0.0",
		},
		{
			name: "nested path",
			input: `spec:
  template:
    spec:
      image:
        tag: v1.0.0
`,
			config:      VersionFileConfig{Path: "spec.template.spec.image.tag"},
			version:     "2.0.0",
			wantContain: "tag: 2.0.0",
		},
		{
			name: "with prefix",
			input: `image:
  tag: v1.0.0
`,
			config:      VersionFileConfig{Path: "image.tag", Prefix: "v"},
			version:     "2.0.0",
			wantContain: "tag: v2.0.0",
		},
		{
			name: "without prefix",
			input: `image:
  tag: 1.0.0
`,
			config:      VersionFileConfig{Path: "image.tag"},
			version:     "2.0.0",
			wantContain: "tag: 2.0.0",
		},
		{
			name: "array index",
			input: `containers:
  - name: app
    image: myapp:v1.0.0
  - name: sidecar
    image: sidecar:v1.0.0
`,
			config:      VersionFileConfig{Path: "containers[0].image"},
			version:     "myapp:v2.0.0",
			wantContain: "image: myapp:v2.0.0",
		},
		{
			name: "top level key",
			input: `version: 1.0.0
name: myapp
`,
			config:      VersionFileConfig{Path: "version"},
			version:     "2.0.0",
			wantContain: "version: 2.0.0",
		},
		{
			name: "key not found",
			input: `metadata:
  name: test
`,
			config:  VersionFileConfig{Path: "metadata.version"},
			version: "2.0.0",
			wantErr: true,
		},
		{
			name: "invalid path - missing parent",
			input: `metadata:
  name: test
`,
			config:  VersionFileConfig{Path: "spec.version"},
			version: "2.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpPath := createTempFile(t, tt.input, "yaml-test-*.yaml")

			cfg := tt.config
			cfg.File = tmpPath

			err := UpdateYAMLFile(cfg, tt.version)
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

	err := UpdateYAMLFile(cfg, "1.0.0")
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

	if err := UpdateYAMLFile(cfg, "2.0.0"); err != nil {
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
		{".image.tag", "", true},           // Leading dot
		{".version", "", true},             // Leading dot
		{"..recursive", "", true},          // Double dot
		{"", "", true},                     // Empty path
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

	err := UpdateYAMLFile(cfg, "2.0.0")
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
				File: tmpPath,
				Path: "image.tag",
			}

			if err := UpdateYAMLFile(cfg, "v2.0.0"); err != nil {
				t.Fatalf("UpdateYAMLFile() error = %v", err)
			}

			got := readTempFile(t, tmpPath)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("UpdateYAMLFile() quote style not preserved, want %q in:\n%s", tt.wantContain, got)
			}
		})
	}
}
