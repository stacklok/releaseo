# Releaseo

A GitHub Action written in Go that creates release PRs with automated version bumping.

## Features

- Semantic version bumping (major, minor, patch)
- Updates `VERSION` file
- Updates Helm chart files (`Chart.yaml`, `values.yaml`)
- Creates release branch and PR automatically
- Validates version is increasing

## Usage

### Basic Usage

```yaml
name: Create Release PR

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
          bump_type: ${{ inputs.bump_type }}
          token: ${{ secrets.GITHUB_TOKEN }}
```

### With Helm Chart

```yaml
- name: Create Release PR
  uses: stacklok/releaseo@v1
  with:
    bump_type: ${{ inputs.bump_type }}
    chart_path: deploy/charts/my-app
    token: ${{ secrets.GITHUB_TOKEN }}
```

### Using Outputs

```yaml
- name: Create Release PR
  id: release
  uses: stacklok/releaseo@v1
  with:
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
| `bump_type` | Version bump type (`major`, `minor`, `patch`) | Yes | - |
| `version_file` | Path to VERSION file | No | `VERSION` |
| `chart_path` | Path to Helm chart directory | No | - |
| `token` | GitHub token for creating PR | Yes | - |
| `base_branch` | Base branch for the PR | No | `main` |

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
5. If `chart_path` provided, updates:
   - `Chart.yaml` (`version` and `appVersion`)
   - `values.yaml` (`image.tag` with `v` prefix)
6. Creates branch `release/v{version}`
7. Commits changes
8. Creates pull request

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
  --chart-path=deploy/charts/myapp
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
