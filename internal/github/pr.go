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
	"fmt"
	"path/filepath"

	"github.com/google/go-github/v60/github"
)

// CreateReleasePR creates a new branch with the modified files and opens a PR.
func (c *Client) CreateReleasePR(ctx context.Context, req PRRequest) (*PRResult, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid PR request: %w", err)
	}

	// Get the base branch reference
	baseRef, _, err := c.client.Git.GetRef(ctx, req.Owner, req.Repo, "refs/heads/"+req.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("getting base branch ref: %w", err)
	}

	// Create the new branch
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + req.HeadBranch),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}

	_, _, err = c.client.Git.CreateRef(ctx, req.Owner, req.Repo, newRef)
	if err != nil {
		return nil, fmt.Errorf("creating branch: %w", err)
	}

	// Commit the files to the new branch
	for _, filePath := range req.Files {
		if err := c.commitFile(ctx, req.Owner, req.Repo, req.HeadBranch, filePath); err != nil {
			return nil, fmt.Errorf("committing file %s: %w", filePath, err)
		}
	}

	// Create the pull request
	pr, _, err := c.client.PullRequests.Create(ctx, req.Owner, req.Repo, &github.NewPullRequest{
		Title: github.String(req.Title),
		Head:  github.String(req.HeadBranch),
		Base:  github.String(req.BaseBranch),
		Body:  github.String(req.Body),
	})
	if err != nil {
		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	// Add release label (non-fatal if it fails, label might not exist)
	_, _, _ = c.client.Issues.AddLabelsToIssue(ctx, req.Owner, req.Repo, pr.GetNumber(), []string{"release"})

	return &PRResult{
		Number: pr.GetNumber(),
		URL:    pr.GetHTMLURL(),
	}, nil
}

// commitFile commits a single file to a branch.
func (c *Client) commitFile(ctx context.Context, owner, repo, branch, filePath string) error {
	// Read file content using the fileReader interface
	content, err := c.fileReader.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Get current file (to get SHA for update)
	existingFile, _, _, err := c.client.Repositories.GetContents(
		ctx, owner, repo, filePath,
		&github.RepositoryContentGetOptions{Ref: branch},
	)

	message := fmt.Sprintf("Update %s for release", filepath.Base(filePath))

	opts := &github.RepositoryContentFileOptions{
		Message: github.String(message),
		Content: content,
		Branch:  github.String(branch),
	}

	if err == nil && existingFile != nil {
		// File exists - update it
		opts.SHA = existingFile.SHA
		_, _, err = c.client.Repositories.UpdateFile(ctx, owner, repo, filePath, opts)
	} else {
		// File doesn't exist - create it
		_, _, err = c.client.Repositories.CreateFile(ctx, owner, repo, filePath, opts)
	}

	if err != nil {
		return fmt.Errorf("updating file: %w", err)
	}

	return nil
}
