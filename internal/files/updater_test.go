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
	"path/filepath"
	"strings"
	"testing"
)

func TestReadVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "basic version",
			content: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "with newline",
			content: "1.2.3\n",
			want:    "1.2.3",
		},
		{
			name:    "with whitespace",
			content: "  1.2.3  \n",
			want:    "1.2.3",
		},
		{
			name:    "with v prefix",
			content: "v1.2.3\n",
			want:    "v1.2.3",
		},
		{
			name:    "empty file",
			content: "",
			wantErr: true,
		},
		{
			name:    "only whitespace",
			content: "   \n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create temp file
			tmpFile, err := os.CreateTemp("", "version-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if err := os.WriteFile(tmpFile.Name(), []byte(tt.content), 0600); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			got, err := ReadVersion(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadVersion_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := ReadVersion("/nonexistent/path/VERSION")
	if err == nil {
		t.Error("ReadVersion() expected error for nonexistent file")
	}
}

func TestWriteVersion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "basic version",
			version: "1.2.3",
			want:    "1.2.3\n",
		},
		{
			name:    "already has newline",
			version: "1.2.3\n",
			want:    "1.2.3\n",
		},
		{
			name:    "with whitespace",
			version: "  1.2.3  ",
			want:    "1.2.3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpFile, err := os.CreateTemp("", "version-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			if err := WriteVersion(tmpFile.Name(), tt.version); err != nil {
				t.Errorf("WriteVersion() error = %v", err)
				return
			}

			got, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("failed to read temp file: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("WriteVersion() wrote %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestUpdateChartYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		version     string
		wantVersion string
		wantApp     string
		wantErr     bool
	}{
		{
			name: "basic update",
			input: `apiVersion: v2
name: test-chart
version: 1.0.0
appVersion: 1.0.0
`,
			version:     "2.0.0",
			wantVersion: "2.0.0",
			wantApp:     "2.0.0",
		},
		{
			name: "preserves other fields",
			input: `apiVersion: v2
name: test-chart
description: A test chart
version: 1.0.0
appVersion: 1.0.0
type: application
`,
			version:     "2.0.0",
			wantVersion: "2.0.0",
			wantApp:     "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir, err := os.MkdirTemp("", "chart-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			chartPath := filepath.Join(tmpDir, "Chart.yaml")
			if err := os.WriteFile(chartPath, []byte(tt.input), 0600); err != nil {
				t.Fatalf("failed to write Chart.yaml: %v", err)
			}

			if err := UpdateChartYAML(tmpDir, tt.version); (err != nil) != tt.wantErr {
				t.Errorf("UpdateChartYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, err := os.ReadFile(chartPath)
			if err != nil {
				t.Fatalf("failed to read Chart.yaml: %v", err)
			}

			content := string(got)
			if !strings.Contains(content, "version: "+tt.wantVersion) {
				t.Errorf("UpdateChartYAML() version not updated, got:\n%s", content)
			}
			if !strings.Contains(content, "appVersion: "+tt.wantApp) {
				t.Errorf("UpdateChartYAML() appVersion not updated, got:\n%s", content)
			}
		})
	}
}

func TestUpdateChartYAML_FileNotFound(t *testing.T) {
	t.Parallel()
	err := UpdateChartYAML("/nonexistent/path", "1.0.0")
	if err == nil {
		t.Error("UpdateChartYAML() expected error for nonexistent file")
	}
}

func TestUpdateChartYAML_MissingField(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "chart-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Chart.yaml missing version field
	input := `apiVersion: v2
name: test-chart
`
	chartPath := filepath.Join(tmpDir, "Chart.yaml")
	if err := os.WriteFile(chartPath, []byte(input), 0600); err != nil {
		t.Fatalf("failed to write Chart.yaml: %v", err)
	}

	err = UpdateChartYAML(tmpDir, "1.0.0")
	if err == nil {
		t.Error("UpdateChartYAML() expected error for missing version field")
	}
}

func TestUpdateValuesYAML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		version string
		wantTag string
		wantErr bool
	}{
		{
			name: "basic update",
			input: `image:
  repository: ghcr.io/stacklok/test
  tag: v1.0.0
`,
			version: "2.0.0",
			wantTag: "v2.0.0",
		},
		{
			name: "preserves other fields",
			input: `replicaCount: 3
image:
  repository: ghcr.io/stacklok/test
  tag: v1.0.0
  pullPolicy: IfNotPresent
service:
  type: ClusterIP
`,
			version: "2.0.0",
			wantTag: "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir, err := os.MkdirTemp("", "chart-*")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			valuesPath := filepath.Join(tmpDir, "values.yaml")
			if err := os.WriteFile(valuesPath, []byte(tt.input), 0600); err != nil {
				t.Fatalf("failed to write values.yaml: %v", err)
			}

			if err := UpdateValuesYAML(tmpDir, tt.version); (err != nil) != tt.wantErr {
				t.Errorf("UpdateValuesYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			got, err := os.ReadFile(valuesPath)
			if err != nil {
				t.Fatalf("failed to read values.yaml: %v", err)
			}

			content := string(got)
			if !strings.Contains(content, "tag: "+tt.wantTag) {
				t.Errorf("UpdateValuesYAML() tag not updated, got:\n%s", content)
			}
		})
	}
}

func TestUpdateValuesYAML_FileNotFound(t *testing.T) {
	t.Parallel()
	err := UpdateValuesYAML("/nonexistent/path", "1.0.0")
	if err == nil {
		t.Error("UpdateValuesYAML() expected error for nonexistent file")
	}
}

func TestUpdateValuesYAML_MissingImageSection(t *testing.T) {
	t.Parallel()
	tmpDir, err := os.MkdirTemp("", "chart-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// values.yaml missing image section
	input := `replicaCount: 3
service:
  type: ClusterIP
`
	valuesPath := filepath.Join(tmpDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte(input), 0600); err != nil {
		t.Fatalf("failed to write values.yaml: %v", err)
	}

	err = UpdateValuesYAML(tmpDir, "1.0.0")
	if err == nil {
		t.Error("UpdateValuesYAML() expected error for missing image section")
	}
}
