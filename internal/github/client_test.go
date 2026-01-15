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
