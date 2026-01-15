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

func TestValidatePath(t *testing.T) {
	t.Parallel()

	// Create a temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		basePath string
		userPath string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid relative path",
			basePath: tempDir,
			userPath: "subdir/file.txt",
			wantErr:  false,
		},
		{
			name:     "valid simple filename",
			basePath: tempDir,
			userPath: "file.txt",
			wantErr:  false,
		},
		{
			name:     "empty path",
			basePath: tempDir,
			userPath: "",
			wantErr:  true,
			errMsg:   "path cannot be empty",
		},
		{
			name:     "path traversal with ..",
			basePath: tempDir,
			userPath: "../etc/passwd",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
		{
			name:     "path traversal with multiple ..",
			basePath: tempDir,
			userPath: "../../etc/passwd",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
		{
			name:     "path traversal in middle",
			basePath: tempDir,
			userPath: "foo/../../../etc/passwd",
			wantErr:  true,
			errMsg:   "path traversal detected",
		},
		{
			name:     "absolute path outside base",
			basePath: tempDir,
			userPath: "/etc/passwd",
			wantErr:  true,
			errMsg:   "resolves outside",
		},
		{
			name:     "valid nested path",
			basePath: tempDir,
			userPath: "deploy/charts/myapp/Chart.yaml",
			wantErr:  false,
		},
		{
			name:     "path with current dir reference",
			basePath: tempDir,
			userPath: "./file.txt",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ValidatePath(tt.basePath, tt.userPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePath() unexpected error = %v", err)
				return
			}

			// Verify result is within base directory
			absBase, _ := filepath.Abs(tt.basePath)
			if !strings.HasPrefix(result, absBase) {
				t.Errorf("ValidatePath() result %q not within base %q", result, absBase)
			}
		})
	}
}

func TestValidatePath_EmptyBasePath(t *testing.T) {
	t.Parallel()

	// Should use current working directory when base is empty
	result, err := ValidatePath("", "file.txt")
	if err != nil {
		t.Errorf("ValidatePath() with empty base unexpected error = %v", err)
		return
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, "file.txt")
	if result != expected {
		t.Errorf("ValidatePath() = %q, want %q", result, expected)
	}
}

func TestValidatePathRelative(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name     string
		basePath string
		userPath string
		want     string
		wantErr  bool
	}{
		{
			name:     "simple relative path",
			basePath: tempDir,
			userPath: "file.txt",
			want:     "file.txt",
			wantErr:  false,
		},
		{
			name:     "nested relative path",
			basePath: tempDir,
			userPath: "subdir/file.txt",
			want:     filepath.Join("subdir", "file.txt"),
			wantErr:  false,
		},
		{
			name:     "path traversal rejected",
			basePath: tempDir,
			userPath: "../file.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ValidatePathRelative(tt.basePath, tt.userPath)

			if tt.wantErr {
				if err == nil {
					t.Error("ValidatePathRelative() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePathRelative() unexpected error = %v", err)
				return
			}

			if result != tt.want {
				t.Errorf("ValidatePathRelative() = %q, want %q", result, tt.want)
			}
		})
	}
}
