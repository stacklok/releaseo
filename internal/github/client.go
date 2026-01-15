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

// Package github provides utilities for interacting with the GitHub API.
package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// PRCreator defines the interface for creating pull requests.
type PRCreator interface {
	// CreateReleasePR creates a new branch with the modified files and opens a PR.
	CreateReleasePR(ctx context.Context, req PRRequest) (*PRResult, error)
}

// Client wraps the GitHub API client and implements PRCreator.
type Client struct {
	client     *github.Client
	fileReader FileReader
}

// Ensure Client implements PRCreator at compile time.
var _ PRCreator = (*Client)(nil)

// osFileReader is the default FileReader implementation that uses os.ReadFile.
type osFileReader struct{}

// ReadFile reads the contents of a file using the standard library os.ReadFile.
func (*osFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ClientOption is a functional option for configuring the Client.
type ClientOption func(*Client)

// WithFileReader sets a custom FileReader implementation for the Client.
// This is useful for testing or when file reading needs to be customized.
func WithFileReader(fr FileReader) ClientOption {
	return func(c *Client) {
		c.fileReader = fr
	}
}

// NewClient creates a new GitHub client with the provided token.
// Optional ClientOption functions can be provided to customize the client behavior.
func NewClient(ctx context.Context, token string, opts ...ClientOption) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	c := &Client{
		client:     github.NewClient(tc),
		fileReader: &osFileReader{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// PRRequest contains the parameters for creating a pull request.
// All fields except Body are required.
type PRRequest struct {
	Owner      string   // GitHub repository owner (required)
	Repo       string   // GitHub repository name (required)
	BaseBranch string   // Base branch for the PR (required, e.g., "main")
	HeadBranch string   // Feature branch to create (required)
	Title      string   // PR title (required)
	Body       string   // PR body/description
	Files      []string // Files to commit (required, must not be empty)
}

// Validate checks that all required fields are set.
func (r *PRRequest) Validate() error {
	if r.Owner == "" {
		return fmt.Errorf("owner is required")
	}
	if r.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if r.BaseBranch == "" {
		return fmt.Errorf("base branch is required")
	}
	if r.HeadBranch == "" {
		return fmt.Errorf("head branch is required")
	}
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Files) == 0 {
		return fmt.Errorf("at least one file is required")
	}
	return nil
}

// PRResult contains the result of creating a pull request.
type PRResult struct {
	Number int
	URL    string
}
