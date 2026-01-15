# Releaseo

A GitHub Action written in Go that creates release PRs with automated version bumping for VERSION files, Helm charts, and any YAML configuration.

## Why Releaseo?

### The Problem

Maintaining version consistency across a project is surprisingly difficult, especially when deploying containerized applications with Helm charts. Common pain points include:

- **Chart versions drift from image versions**: When you release a new version of your application, the Helm chart's `appVersion`, `image.tag`, and chart `version` need to stay in sync. This is often forgotten or done inconsistently.

- **Manual version bumps are error-prone**: Developers frequently forget to update all the places where versions live, leading to mismatched deployments.

- **Release automation gaps**: Tools like Renovate handle dependency updates well, but don't solve the problem of coordinating your own application's version across multiple files.

- **Documentation gets out of sync**: Helm chart documentation (via helm-docs) needs regenerating when versions change, but this step is often missed.

This problem is well-documented in projects like [ToolHive](https://github.com/stacklok/toolhive/issues/1779), where the release flow required manual intervention to bump chart versions after image releases, causing CI failures and forgotten updates.

### The Solution

Releaseo automates the entire version bump workflow:

1. Bump the version in your `VERSION` file (single source of truth)
2. Propagate that version to any YAML files you specify (Chart.yaml, values.yaml, etc.)
3. Optionally run helm-docs to regenerate chart documentation
4. Create a release PR with all changes ready for review

No more forgotten version bumps. No more mismatched chart and image versions.

## Features

- Semantic version bumping (major, minor, patch)
- Updates `VERSION` file as single source of truth
- Updates any YAML file at any path (Chart.yaml version, appVersion, values.yaml image.tag, etc.)
- Optional helm-docs integration for chart documentation
- Creates release branch and PR automatically
- Validates version is increasing
- Preserves YAML formatting and comments

## Usage

### Basic Usage

```yaml
name: Release

on:
  workflow_dispatch:
    inputs:
      bump_type:
        description: 'Version bump type'
        required: true
        type: choice
        options:
          - patch
          - minor
          - major

permissions:
  contents: write
  pull-requests: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Create Release PR
        uses: stacklok/releaseo@v1
        with:
          releaseo_version: v1.0.0
          bump_type: ${{ inputs.bump_type }}
          token: ${{ secrets.GITHUB_TOKEN }}
```

### With Helm Chart Version Sync

```yaml
- name: Create Release PR
  uses: stacklok/releaseo@v1
  with:
    releaseo_version: v1.0.0
    bump_type: ${{ inputs.bump_type }}
    token: ${{ secrets.GITHUB_TOKEN }}
    version_files: |
      - file: deploy/charts/myapp/Chart.yaml
        path: version
      - file: deploy/charts/myapp/Chart.yaml
        path: appVersion
      - file: deploy/charts/myapp/values.yaml
        path: image.tag
        prefix: "v"
```

### With helm-docs Integration

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.23'

- name: Install helm-docs
  run: go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest

- name: Create Release PR
  uses: stacklok/releaseo@v1
  with:
    releaseo_version: v1.0.0
    bump_type: ${{ inputs.bump_type }}
    token: ${{ secrets.GITHUB_TOKEN }}
    version_files: |
      - file: deploy/charts/myapp/Chart.yaml
        path: version
      - file: deploy/charts/myapp/Chart.yaml
        path: appVersion
    helm_docs_args: --chart-search-root=deploy/charts/myapp --template-files=README.md.gotmpl
```

### Using Outputs

```yaml
- name: Create Release PR
  id: release
  uses: stacklok/releaseo@v1
  with:
    releaseo_version: v1.0.0
    bump_type: ${{ inputs.bump_type }}
    token: ${{ secrets.GITHUB_TOKEN }}

- name: Print PR URL
  run: |
    echo "Created PR #${{ steps.release.outputs.pr_number }}"
    echo "URL: ${{ steps.release.outputs.pr_url }}"
    echo "New version: ${{ steps.release.outputs.version }}"
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `releaseo_version` | Version of releaseo to use (e.g., `v1.0.0`) | Yes | - |
| `bump_type` | Version bump type (`major`, `minor`, `patch`) | Yes | - |
| `version_file` | Path to VERSION file | No | `VERSION` |
| `version_files` | YAML list of files with paths to update (see below) | No | - |
| `helm_docs_args` | Arguments to pass to helm-docs (if provided, helm-docs runs) | No | - |
| `token` | GitHub token for creating PR | Yes | - |
| `base_branch` | Base branch for the PR | No | `main` |

### version_files Format

The `version_files` input accepts a YAML list where each entry specifies:
- `file`: Path to the YAML file
- `path`: Dot-notation path to the value (e.g., `image.tag`, `metadata.version`)
- `prefix`: Optional prefix to prepend to the version (e.g., `v` for `v1.0.0`)

```yaml
version_files: |
  - file: deploy/charts/myapp/Chart.yaml
    path: version
  - file: deploy/charts/myapp/Chart.yaml
    path: appVersion
  - file: deploy/charts/myapp/values.yaml
    path: image.tag
    prefix: "v"
  - file: config/version.yaml
    path: spec.version
```

## Outputs

| Output | Description |
|--------|-------------|
| `version` | The new version number |
| `pr_number` | The created PR number |
| `pr_url` | The created PR URL |

## How It Works

1. Reads current version from `VERSION` file
2. Calculates new version based on bump type:
   - `major`: `1.0.0` → `2.0.0`
   - `minor`: `1.0.0` → `1.1.0`
   - `patch`: `1.0.0` → `1.0.1`
3. Validates new version is greater than current
4. Updates `VERSION` file
5. Updates all specified `version_files` at their configured paths
6. Runs helm-docs if `helm_docs_args` is provided
7. Creates branch `release/v{version}`
8. Commits all changes
9. Creates pull request with release label

## Development

### Building

```bash
go build -o releaseo .
```

### Testing

```bash
go test ./...
```

### Running Locally

```bash
export GITHUB_TOKEN=your_token
export GITHUB_REPOSITORY=owner/repo

./releaseo \
  --bump-type=patch \
  --version-file=VERSION \
  --version-files='[{"file":"deploy/charts/myapp/Chart.yaml","path":"version"}]'
```

## Related Issues

- [ToolHive: Better Chart Release Flow](https://github.com/stacklok/toolhive/issues/1779) - The original motivation for this tool

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
