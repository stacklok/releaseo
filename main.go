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
	BumpType     string
	VersionFile  string
	HelmDocsArgs string
	VersionFiles []files.VersionFileConfig
	Token        string
	RepoOwner    string
	RepoName     string
	BaseBranch   string
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
	// Bump version
	newVersion, err := bumpVersion(cfg)
	if err != nil {
		return err
	}

	// Update all files
	helmDocsFiles := updateAllFiles(cfg, newVersion.String())

	// Create the release PR
	pr, err := createReleasePR(ctx, cfg, newVersion.String(), helmDocsFiles)
	if err != nil {
		return err
	}

	// Set GitHub Actions outputs
	setOutput("version", newVersion.String())
	setOutput("pr_number", fmt.Sprintf("%d", pr.Number))
	setOutput("pr_url", pr.URL)

	return nil
}

// bumpVersion reads the current version and bumps it according to the bump type.
func bumpVersion(cfg Config) (*version.Version, error) {
	currentVersion, err := files.ReadVersion(cfg.VersionFile)
	if err != nil {
		return nil, fmt.Errorf("reading version: %w", err)
	}
	fmt.Printf("Current version: %s\n", currentVersion)

	v, err := version.Parse(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing version: %w", err)
	}

	newVersion, err := v.Bump(cfg.BumpType)
	if err != nil {
		return nil, fmt.Errorf("bumping version: %w", err)
	}
	fmt.Printf("New version: %s (%s bump)\n", newVersion, cfg.BumpType)

	if !version.IsGreater(newVersion.String(), currentVersion) {
		return nil, fmt.Errorf("new version %s is not greater than current %s", newVersion, currentVersion)
	}

	return newVersion, nil
}

// updateAllFiles updates the VERSION file, custom version files, and runs helm-docs.
// Returns the list of files modified by helm-docs.
func updateAllFiles(cfg Config, newVersion string) []string {
	// Update VERSION file
	if err := files.WriteVersion(cfg.VersionFile, newVersion); err != nil {
		fmt.Printf("Warning: could not write version file: %v\n", err)
	} else {
		fmt.Printf("Updated %s\n", cfg.VersionFile)
	}

	// Update custom version files
	for _, vf := range cfg.VersionFiles {
		if err := files.UpdateYAMLFile(vf, newVersion); err != nil {
			fmt.Printf("Warning: could not update %s at %s: %v\n", vf.File, vf.Path, err)
		} else {
			fmt.Printf("Updated %s at path %s\n", vf.File, vf.Path)
		}
	}

	// Run helm-docs if args are provided
	var helmDocsFiles []string
	if cfg.HelmDocsArgs != "" {
		var err error
		helmDocsFiles, err = runHelmDocs(cfg.HelmDocsArgs)
		if err != nil {
			fmt.Printf("Warning: could not run helm-docs: %v\n", err)
		} else {
			fmt.Printf("Ran helm-docs successfully\n")
			if len(helmDocsFiles) > 0 {
				fmt.Printf("Files modified by helm-docs: %v\n", helmDocsFiles)
			}
		}
	}

	return helmDocsFiles
}

// createReleasePR creates the GitHub release PR with all modified files.
func createReleasePR(ctx context.Context, cfg Config, newVersion string, helmDocsFiles []string) (*github.PRResult, error) {
	gh, err := github.NewClient(ctx, cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub client: %w", err)
	}

	branchName := fmt.Sprintf("release/v%s", newVersion)
	prTitle := fmt.Sprintf("Release v%s", newVersion)
	prBody := generatePRBody(newVersion, cfg.BumpType, cfg.VersionFiles, cfg.HelmDocsArgs != "")

	allFiles := getModifiedFiles(cfg)
	allFiles = append(allFiles, helmDocsFiles...)

	pr, err := gh.CreateReleasePR(ctx, github.PRRequest{
		Owner:      cfg.RepoOwner,
		Repo:       cfg.RepoName,
		BaseBranch: cfg.BaseBranch,
		HeadBranch: branchName,
		Title:      prTitle,
		Body:       prBody,
		Files:      allFiles,
	})
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	fmt.Printf("\nRelease PR created: %s\n", pr.URL)
	return pr, nil
}

func parseFlags() Config {
	cfg := Config{}
	var versionFilesJSON string

	flag.StringVar(&cfg.BumpType, "bump-type", "", "Version bump type (major, minor, patch)")
	flag.StringVar(&cfg.VersionFile, "version-file", "VERSION", "Path to VERSION file")
	flag.StringVar(&cfg.HelmDocsArgs, "helm-docs-args", "", "Arguments to pass to helm-docs (if provided, helm-docs will run)")
	flag.StringVar(&versionFilesJSON, "version-files", "", "JSON array of {file, path, prefix} objects for custom version updates")
	flag.StringVar(&cfg.Token, "token", "", "GitHub token")
	flag.StringVar(&cfg.BaseBranch, "base-branch", "main", "Base branch for PR")
	flag.Parse()

	cfg.VersionFiles = parseVersionFiles(versionFilesJSON)
	cfg.Token = resolveToken(cfg.Token)
	cfg.RepoOwner, cfg.RepoName = parseRepository()

	validateConfig(cfg)

	return cfg
}

// parseVersionFiles parses the JSON array of version file configurations.
func parseVersionFiles(jsonStr string) []files.VersionFileConfig {
	if jsonStr == "" {
		return nil
	}

	var versionFiles []files.VersionFileConfig
	if err := json.Unmarshal([]byte(jsonStr), &versionFiles); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing --version-files JSON: %v\n", err)
		os.Exit(1)
	}
	return versionFiles
}

// resolveToken returns the token from the flag or environment variable.
func resolveToken(flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	return os.Getenv("GITHUB_TOKEN")
}

// parseRepository extracts owner and repo from GITHUB_REPOSITORY environment variable.
func parseRepository() (owner, repo string) {
	repoEnv := os.Getenv("GITHUB_REPOSITORY")
	if repoEnv == "" {
		return "", ""
	}

	parts := strings.Split(repoEnv, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// validateConfig ensures all required configuration fields are set.
func validateConfig(cfg Config) {
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
}

func generatePRBody(ver, bumpType string, versionFiles []files.VersionFileConfig, ranHelmDocs bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Release v%s\n\n", ver))
	sb.WriteString("### Version Bump\n\n")
	sb.WriteString(fmt.Sprintf("**%s** release\n\n", bumpType))
	sb.WriteString("### Files Updated\n\n")
	sb.WriteString("- `VERSION`\n")

	for _, vf := range versionFiles {
		sb.WriteString(fmt.Sprintf("- `%s` (path: `%s`)\n", vf.File, vf.Path))
	}

	if ranHelmDocs {
		sb.WriteString("- Helm chart docs (via helm-docs)\n")
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
	return modifiedFiles
}

// runHelmDocs executes helm-docs with the provided arguments and returns the list of modified files.
func runHelmDocs(argsStr string) ([]string, error) {
	args := strings.Fields(argsStr)
	cmd := exec.Command("helm-docs", args...) //nolint:gosec // args are from trusted input
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Detect files modified by helm-docs using git
	return getGitModifiedFiles()
}

// getGitModifiedFiles returns a list of files that have been modified in the working directory.
func getGitModifiedFiles() ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running git status: %w", err)
	}

	var result []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// git status --porcelain format: "XY filename" where XY is the status
		// We want files that are modified (M) or added (A) in the working tree
		if len(line) >= 3 {
			file := strings.TrimSpace(line[2:])
			if file != "" {
				result = append(result, file)
			}
		}
	}
	return result, nil
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
