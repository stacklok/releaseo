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

package github

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "ghp_validtoken123",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(context.Background(), tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
		})
	}
}

func TestPRRequest_Validate(t *testing.T) {
	t.Parallel()

	validRequest := PRRequest{
		Owner:      "owner",
		Repo:       "repo",
		BaseBranch: "main",
		HeadBranch: "release/v1.0.0",
		Title:      "Release v1.0.0",
		Body:       "Release body",
		Files:      []string{"VERSION"},
	}

	tests := []struct {
		name    string
		modify  func(*PRRequest)
		wantErr string
	}{
		{
			name:    "valid request",
			modify:  func(_ *PRRequest) {},
			wantErr: "",
		},
		{
			name:    "missing owner",
			modify:  func(r *PRRequest) { r.Owner = "" },
			wantErr: "owner is required",
		},
		{
			name:    "missing repo",
			modify:  func(r *PRRequest) { r.Repo = "" },
			wantErr: "repo is required",
		},
		{
			name:    "missing base branch",
			modify:  func(r *PRRequest) { r.BaseBranch = "" },
			wantErr: "base branch is required",
		},
		{
			name:    "missing head branch",
			modify:  func(r *PRRequest) { r.HeadBranch = "" },
			wantErr: "head branch is required",
		},
		{
			name:    "missing title",
			modify:  func(r *PRRequest) { r.Title = "" },
			wantErr: "title is required",
		},
		{
			name:    "missing files",
			modify:  func(r *PRRequest) { r.Files = nil },
			wantErr: "at least one file is required",
		},
		{
			name:    "empty files slice",
			modify:  func(r *PRRequest) { r.Files = []string{} },
			wantErr: "at least one file is required",
		},
		{
			name:    "body is optional",
			modify:  func(r *PRRequest) { r.Body = "" },
			wantErr: "",
		},
		{
			name:    "triggered by is optional",
			modify:  func(r *PRRequest) { r.TriggeredBy = "" },
			wantErr: "",
		},
		{
			name:    "triggered by with value",
			modify:  func(r *PRRequest) { r.TriggeredBy = "someuser" },
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := validRequest
			tt.modify(&req)

			err := req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

// mockFileReader is a simple mock implementation for testing FileReader injection.
type mockFileReader struct {
	called bool
}

func (m *mockFileReader) ReadFile(_ string) ([]byte, error) {
	m.called = true
	return []byte("mock content"), nil
}

func TestWithFileReader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fileReader FileReader
	}{
		{
			name:       "custom FileReader is injected",
			fileReader: &mockFileReader{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(context.Background(), "test-token", WithFileReader(tt.fileReader))
			if err != nil {
				t.Fatalf("NewClient() unexpected error = %v", err)
			}
			if client.fileReader != tt.fileReader {
				t.Error("WithFileReader() did not inject the custom FileReader")
			}
		})
	}
}

func TestClient_ImplementsPRCreator(t *testing.T) {
	t.Parallel()

	client, err := NewClient(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("NewClient() unexpected error = %v", err)
	}

	// Runtime assertion that Client implements PRCreator interface.
	var _ PRCreator = client
}

// TestDeduplicateFiles tests that duplicate file paths are removed while preserving order.
func TestDeduplicateFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "no duplicates",
			input: []string{"a.yaml", "b.yaml", "c.yaml"},
			want:  []string{"a.yaml", "b.yaml", "c.yaml"},
		},
		{
			name:  "single file duplicated",
			input: []string{"deploy/charts/operator-crds/Chart.yaml", "deploy/charts/operator-crds/Chart.yaml"},
			want:  []string{"deploy/charts/operator-crds/Chart.yaml"},
		},
		{
			name:  "multiple files with duplicates preserves order",
			input: []string{"deploy/charts/operator/Chart.yaml", "deploy/charts/operator/values.yaml", "deploy/charts/operator/Chart.yaml", "deploy/charts/operator/values.yaml", "deploy/charts/operator/values.yaml"},
			want:  []string{"deploy/charts/operator/Chart.yaml", "deploy/charts/operator/values.yaml"},
		},
		{
			name:  "multiple files with duplicate names but separate paths preserves order",
			input: []string{"deploy/charts/operator/Chart.yaml", "deploy/charts/operator/values.yaml", "deploy/charts/operator-crds/Chart.yaml", "deploy/charts/operator-crds/values.yaml"},
			want:  []string{"deploy/charts/operator/Chart.yaml", "deploy/charts/operator/values.yaml", "deploy/charts/operator-crds/Chart.yaml", "deploy/charts/operator-crds/values.yaml"},
		},
		{
			name:  "single file",
			input: []string{"VERSION"},
			want:  []string{"VERSION"},
		},
		{
			name:  "empty slice",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "nil slice",
			input: nil,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := deduplicateFiles(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("deduplicateFiles() returned %d items, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("deduplicateFiles()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestCommitMessageFormat tests the commit message format with and without git trailer.
// This tests the format logic used in commitFiles().
func TestCommitMessageFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		triggeredBy string
		wantMessage string
	}{
		{
			name:        "without triggered by",
			triggeredBy: "",
			wantMessage: "Update release files",
		},
		{
			name:        "with triggered by",
			triggeredBy: "testuser",
			wantMessage: "Update release files\n\nRelease-Triggered-By: testuser",
		},
		{
			name:        "with triggered by from releasebot",
			triggeredBy: "releasebot",
			wantMessage: "Update release files\n\nRelease-Triggered-By: releasebot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Replicate the message format logic from commitFiles()
			message := "Update release files"
			if tt.triggeredBy != "" {
				message += "\n\nRelease-Triggered-By: " + tt.triggeredBy
			}

			if message != tt.wantMessage {
				t.Errorf("commit message = %q, want %q", message, tt.wantMessage)
			}
		})
	}
}
