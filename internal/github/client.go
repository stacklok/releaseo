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

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client.
type Client struct {
	client *github.Client
}

// NewClient creates a new GitHub client with the provided token.
func NewClient(ctx context.Context, token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
	}, nil
}

// PRRequest contains the parameters for creating a pull request.
type PRRequest struct {
	Owner      string
	Repo       string
	BaseBranch string
	HeadBranch string
	Title      string
	Body       string
	Files      []string
}

// PRResult contains the result of creating a pull request.
type PRResult struct {
	Number int
	URL    string
}
