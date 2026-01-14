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
	"os"
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

			// Create temp file
			tmpFile, err := os.CreateTemp("", "yaml-test-*.yaml")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if err := os.WriteFile(tmpFile.Name(), []byte(tt.input), 0600); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			// Set the file path in config
			cfg := tt.config
			cfg.File = tmpFile.Name()

			// Run the update
			err = UpdateYAMLFile(cfg, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateYAMLFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Read the result
			got, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("failed to read temp file: %v", err)
			}

			if !strings.Contains(string(got), tt.wantContain) {
				t.Errorf("UpdateYAMLFile() result does not contain %q, got:\n%s", tt.wantContain, string(got))
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

func TestParsePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		want    []pathPart
		wantErr bool
	}{
		{
			name: "simple key",
			path: "version",
			want: []pathPart{{Key: "version", Index: -1}},
		},
		{
			name: "dotted path",
			path: "metadata.version",
			want: []pathPart{
				{Key: "metadata", Index: -1},
				{Key: "version", Index: -1},
			},
		},
		{
			name: "deep path",
			path: "spec.template.spec.image.tag",
			want: []pathPart{
				{Key: "spec", Index: -1},
				{Key: "template", Index: -1},
				{Key: "spec", Index: -1},
				{Key: "image", Index: -1},
				{Key: "tag", Index: -1},
			},
		},
		{
			name: "array index",
			path: "containers[0].image",
			want: []pathPart{
				{Key: "containers", Index: 0},
				{Key: "image", Index: -1},
			},
		},
		{
			name: "multiple array indices",
			path: "spec.containers[0].ports[1].containerPort",
			want: []pathPart{
				{Key: "spec", Index: -1},
				{Key: "containers", Index: 0},
				{Key: "ports", Index: 1},
				{Key: "containerPort", Index: -1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parsePath() got %d parts, want %d", len(got), len(tt.want))
				return
			}

			for i, part := range got {
				if part.Key != tt.want[i].Key || part.Index != tt.want[i].Index {
					t.Errorf("parsePath() part[%d] = {%s, %d}, want {%s, %d}",
						i, part.Key, part.Index, tt.want[i].Key, tt.want[i].Index)
				}
			}
		})
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

	tmpFile, err := os.CreateTemp("", "yaml-test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := os.WriteFile(tmpFile.Name(), []byte(input), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	cfg := VersionFileConfig{
		File: tmpFile.Name(),
		Path: "data.version",
	}

	if err := UpdateYAMLFile(cfg, "2.0.0"); err != nil {
		t.Fatalf("UpdateYAMLFile() error = %v", err)
	}

	got, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	content := string(got)

	// Check version was updated
	if !strings.Contains(content, "version: 2.0.0") {
		t.Errorf("version not updated, got:\n%s", content)
	}

	// Check other fields preserved
	if !strings.Contains(content, "kind: ConfigMap") {
		t.Errorf("kind field lost, got:\n%s", content)
	}
	if !strings.Contains(content, "app: myapp") {
		t.Errorf("labels lost, got:\n%s", content)
	}
}
