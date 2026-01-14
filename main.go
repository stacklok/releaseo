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

// Package main provides a GitHub Action for creating release PRs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/stacklok/releaseo/internal/files"
	"github.com/stacklok/releaseo/internal/github"
	"github.com/stacklok/releaseo/internal/version"
)

// Config holds the action configuration.
type Config struct {
	BumpType       string
	VersionFile    string
	HelmDocsCharts []string
	HelmDocsArgs   string
	VersionFiles   []files.VersionFileConfig
	Token          string
	RepoOwner      string
	RepoName       string
	BaseBranch     string
}

func main() {
	ctx := context.Background()
	cfg := parseFlags()

	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg Config) error {
	// Read current version
	currentVersion, err := files.ReadVersion(cfg.VersionFile)
	if err != nil {
		return fmt.Errorf("reading version: %w", err)
	}
	fmt.Printf("Current version: %s\n", currentVersion)

	// Parse and bump version
	v, err := version.Parse(currentVersion)
	if err != nil {
		return fmt.Errorf("parsing version: %w", err)
	}

	newVersion, err := v.Bump(cfg.BumpType)
	if err != nil {
		return fmt.Errorf("bumping version: %w", err)
	}
	fmt.Printf("New version: %s (%s bump)\n", newVersion, cfg.BumpType)

	// Validate version is increasing
	if !version.IsGreater(newVersion.String(), currentVersion) {
		return fmt.Errorf("new version %s is not greater than current %s", newVersion, currentVersion)
	}

	// Update VERSION file
	if err := files.WriteVersion(cfg.VersionFile, newVersion.String()); err != nil {
		return fmt.Errorf("writing version: %w", err)
	}
	fmt.Printf("Updated %s\n", cfg.VersionFile)

	// Update custom version files
	for _, vf := range cfg.VersionFiles {
		if err := files.UpdateYAMLFile(vf, newVersion.String()); err != nil {
			fmt.Printf("Warning: could not update %s at %s: %v\n", vf.File, vf.Path, err)
		} else {
			fmt.Printf("Updated %s at path %s\n", vf.File, vf.Path)
		}
	}

	// Run helm-docs for specified charts
	for _, chartPath := range cfg.HelmDocsCharts {
		if err := runHelmDocs(chartPath, cfg.HelmDocsArgs); err != nil {
			fmt.Printf("Warning: could not run helm-docs for %s: %v\n", chartPath, err)
		} else {
			fmt.Printf("Updated %s/README.md via helm-docs\n", chartPath)
		}
	}

	// Create GitHub client
	gh, err := github.NewClient(ctx, cfg.Token)
	if err != nil {
		return fmt.Errorf("creating GitHub client: %w", err)
	}

	// Create branch, commit, and PR
	branchName := fmt.Sprintf("release/v%s", newVersion)
	prTitle := fmt.Sprintf("Release v%s", newVersion)
	prBody := generatePRBody(newVersion.String(), cfg.BumpType, cfg.VersionFiles, cfg.HelmDocsCharts)

	pr, err := gh.CreateReleasePR(ctx, github.PRRequest{
		Owner:      cfg.RepoOwner,
		Repo:       cfg.RepoName,
		BaseBranch: cfg.BaseBranch,
		HeadBranch: branchName,
		Title:      prTitle,
		Body:       prBody,
		Files:      getModifiedFiles(cfg),
	})
	if err != nil {
		return fmt.Errorf("creating PR: %w", err)
	}

	fmt.Printf("\nRelease PR created: %s\n", pr.URL)

	// Set GitHub Actions outputs
	setOutput("version", newVersion.String())
	setOutput("pr_number", fmt.Sprintf("%d", pr.Number))
	setOutput("pr_url", pr.URL)

	return nil
}

func parseFlags() Config {
	cfg := Config{}
	var versionFilesJSON string
	var helmDocsCharts string

	flag.StringVar(&cfg.BumpType, "bump-type", "", "Version bump type (major, minor, patch)")
	flag.StringVar(&cfg.VersionFile, "version-file", "VERSION", "Path to VERSION file")
	flag.StringVar(&helmDocsCharts, "helm-docs", "", "Comma-separated list of chart paths to run helm-docs on")
	flag.StringVar(&cfg.HelmDocsArgs, "helm-docs-args", "", "Additional arguments to pass to helm-docs")
	flag.StringVar(&versionFilesJSON, "version-files", "", "JSON array of {file, path, prefix} objects for custom version updates")
	flag.StringVar(&cfg.Token, "token", "", "GitHub token")
	flag.StringVar(&cfg.BaseBranch, "base-branch", "main", "Base branch for PR")
	flag.Parse()

	// Parse helm-docs chart paths
	if helmDocsCharts != "" {
		for _, chart := range strings.Split(helmDocsCharts, ",") {
			chart = strings.TrimSpace(chart)
			if chart != "" {
				cfg.HelmDocsCharts = append(cfg.HelmDocsCharts, chart)
			}
		}
	}

	// Parse version files JSON if provided
	if versionFilesJSON != "" {
		if err := json.Unmarshal([]byte(versionFilesJSON), &cfg.VersionFiles); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing --version-files JSON: %v\n", err)
			os.Exit(1)
		}
	}

	// Get token from environment if not provided
	if cfg.Token == "" {
		cfg.Token = os.Getenv("GITHUB_TOKEN")
	}

	// Parse repository from GITHUB_REPOSITORY environment variable
	if repo := os.Getenv("GITHUB_REPOSITORY"); repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			cfg.RepoOwner = parts[0]
			cfg.RepoName = parts[1]
		}
	}

	// Validate required fields
	if cfg.BumpType == "" {
		fmt.Fprintln(os.Stderr, "Error: --bump-type is required")
		flag.Usage()
		os.Exit(1)
	}

	if cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "Error: --token or GITHUB_TOKEN is required")
		flag.Usage()
		os.Exit(1)
	}

	if cfg.RepoOwner == "" || cfg.RepoName == "" {
		fmt.Fprintln(os.Stderr, "Error: GITHUB_REPOSITORY environment variable is required")
		os.Exit(1)
	}

	return cfg
}

func generatePRBody(ver, bumpType string, versionFiles []files.VersionFileConfig, helmDocsCharts []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Release v%s\n\n", ver))
	sb.WriteString("### Version Bump\n\n")
	sb.WriteString(fmt.Sprintf("**%s** release\n\n", bumpType))
	sb.WriteString("### Files Updated\n\n")
	sb.WriteString("- `VERSION`\n")

	for _, vf := range versionFiles {
		sb.WriteString(fmt.Sprintf("- `%s` (path: `%s`)\n", vf.File, vf.Path))
	}

	for _, chartPath := range helmDocsCharts {
		sb.WriteString(fmt.Sprintf("- `%s/README.md` (via helm-docs)\n", chartPath))
	}

	sb.WriteString("\n### Next Steps\n\n")
	sb.WriteString("1. Review this PR\n")
	sb.WriteString("2. Merge to main\n")
	sb.WriteString("3. Release automation will handle the rest\n")
	sb.WriteString("\n### Checklist\n\n")
	sb.WriteString("- [ ] Version bump is correct\n")
	sb.WriteString("- [ ] All CI checks pass\n")

	return sb.String()
}

func getModifiedFiles(cfg Config) []string {
	modifiedFiles := []string{cfg.VersionFile}
	for _, vf := range cfg.VersionFiles {
		modifiedFiles = append(modifiedFiles, vf.File)
	}
	for _, chartPath := range cfg.HelmDocsCharts {
		modifiedFiles = append(modifiedFiles, chartPath+"/README.md")
	}
	return modifiedFiles
}

// runHelmDocs executes helm-docs for the specified chart directory.
func runHelmDocs(chartPath, extraArgs string) error {
	args := []string{"--chart-search-root", chartPath}

	// Parse and append extra arguments
	if extraArgs != "" {
		// Split by spaces, but respect quoted strings
		extraArgsList := strings.Fields(extraArgs)
		args = append(args, extraArgsList...)
	}

	cmd := exec.Command("helm-docs", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func setOutput(name, value string) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		fmt.Printf("Output %s=%s\n", name, value)
		return
	}

	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("Warning: could not write output %s: %v\n", name, err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%s=%s\n", name, value)
}
