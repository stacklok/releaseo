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

	// Deduplicate files - each file already has all YAML path changes applied
	// on disk, so committing the same file twice causes 409 conflicts due to
	// GitHub API eventual consistency with sequential SHA updates.
	uniqueFiles := deduplicateFiles(req.Files)

	// Commit all files to the new branch in a single atomic commit
	if err := c.commitFiles(ctx, req.Owner, req.Repo, req.HeadBranch, uniqueFiles, req.TriggeredBy); err != nil {
		return nil, fmt.Errorf("committing files: %w", err)
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

// deduplicateFiles returns a new slice with duplicate file paths removed,
// preserving the order of first occurrence.
func deduplicateFiles(files []string) []string {
	seen := make(map[string]bool, len(files))
	unique := make([]string, 0, len(files))
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique
}

// commitFiles commits all files to a branch in a single atomic commit using the Git Data API.
// If triggeredBy is non-empty, a git trailer is added to the commit message.
func (c *Client) commitFiles(ctx context.Context, owner, repo, branch string, files []string, triggeredBy string) error {
	// Get the current branch reference
	ref, _, err := c.client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("getting branch ref: %w", err)
	}

	// Get the commit to find the base tree
	baseCommit, _, err := c.client.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return fmt.Errorf("getting base commit: %w", err)
	}

	// Build tree entries for all files
	entries := make([]*github.TreeEntry, 0, len(files))
	for _, filePath := range files {
		content, err := c.fileReader.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", filePath, err)
		}
		contentStr := string(content)
		entries = append(entries, &github.TreeEntry{
			Path:    github.String(filePath),
			Mode:    github.String("100644"),
			Type:    github.String("blob"),
			Content: github.String(contentStr),
		})
	}

	// Create a new tree with all file changes
	tree, _, err := c.client.Git.CreateTree(ctx, owner, repo, baseCommit.GetTree().GetSHA(), entries)
	if err != nil {
		return fmt.Errorf("creating tree: %w", err)
	}

	// Build commit message
	message := "Update release files"
	if triggeredBy != "" {
		message += fmt.Sprintf("\n\nRelease-Triggered-By: %s", triggeredBy)
	}

	// Create the commit
	commit, _, err := c.client.Git.CreateCommit(ctx, owner, repo,
		&github.Commit{
			Message: github.String(message),
			Tree:    tree,
			Parents: []*github.Commit{baseCommit},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	// Update the branch reference to point to the new commit
	ref.Object.SHA = commit.SHA
	_, _, err = c.client.Git.UpdateRef(ctx, owner, repo, ref, false)
	if err != nil {
		return fmt.Errorf("updating ref: %w", err)
	}

	return nil
}
