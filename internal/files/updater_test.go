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
			tmpPath := createTempFile(t, tt.content, "version-*")

			got, err := ReadVersion(tmpPath)
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
			tmpPath := createTempFile(t, "", "version-*")

			if err := WriteVersion(tmpPath, tt.version); err != nil {
				t.Errorf("WriteVersion() error = %v", err)
				return
			}

			got := readTempFile(t, tmpPath)
			if got != tt.want {
				t.Errorf("WriteVersion() wrote %q, want %q", got, tt.want)
			}
		})
	}
}
