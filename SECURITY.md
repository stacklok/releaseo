# Security Threat Model - Releaseo GitHub Action

This document describes the threat model for the Releaseo GitHub Action, which automates version bumping and release PR creation.

## Overview

Releaseo is a composite GitHub Action that:
1. Reads a VERSION file and bumps the semantic version
2. Updates version references in YAML files (e.g., Helm charts)
3. Optionally runs `helm-docs` to regenerate documentation
4. Creates a release branch and pull request via the GitHub API

## Trust Boundaries

```
┌─────────────────────────────────────────────────────────────────────┐
│                        GitHub Actions Runner                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    Workflow Environment                        │  │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐   │  │
│  │  │   Inputs    │───▶│  Releaseo   │───▶│  GitHub API     │   │  │
│  │  │ (untrusted) │    │   Action    │    │  (authenticated)│   │  │
│  │  └─────────────┘    └──────┬──────┘    └─────────────────┘   │  │
│  │                            │                                   │  │
│  │                     ┌──────▼──────┐                           │  │
│  │                     │ File System │                           │  │
│  │                     │ (repo clone)│                           │  │
│  │                     └─────────────┘                           │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Trust Boundary 1: Action Inputs → Action Code
- **Boundary**: User-provided workflow inputs enter the action
- **Risk Level**: High - inputs are untrusted and could be malicious
- **Controls**: Input validation, allowlists, environment variable usage

### Trust Boundary 2: Action Code → File System
- **Boundary**: Action reads/writes files in the repository
- **Risk Level**: Medium - path traversal could access unintended files
- **Controls**: Path validation, working directory restrictions

### Trust Boundary 3: Action Code → External Commands
- **Boundary**: Action executes `helm-docs` binary
- **Risk Level**: High - command injection risk
- **Controls**: Argument allowlist validation

### Trust Boundary 4: Action Code → GitHub API
- **Boundary**: Action authenticates to GitHub and creates branches/PRs
- **Risk Level**: Medium - token scope determines blast radius
- **Controls**: Minimum required token permissions

## Data Flow

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Workflow    │     │   action.yml │     │    main.go   │
│  (caller)    │────▶│   (shell)    │────▶│   (binary)   │
└──────────────┘     └──────────────┘     └──────────────┘
      │                     │                     │
      │ bump_type           │ INPUT_BUMP_TYPE     │ Config.BumpType
      │ version_file        │ INPUT_VERSION_FILE  │ Config.VersionFile
      │ helm_docs_args      │ INPUT_HELM_DOCS_ARGS│ Config.HelmDocsArgs
      │ version_files       │ VERSION_FILES_YAML  │ Config.VersionFiles
      │ token               │ GITHUB_TOKEN        │ Config.Token
      │ base_branch         │ INPUT_BASE_BRANCH   │ Config.BaseBranch
      ▼                     ▼                     ▼
```

## Assets

| Asset | Description | Sensitivity |
|-------|-------------|-------------|
| GITHUB_TOKEN | Authentication token for GitHub API | Critical |
| Repository files | Source code and configuration files | High |
| VERSION file | Current semantic version | Low |
| YAML configs | Helm charts, values files | Medium |
| Git history | Commit and branch state | Medium |

## Threat Actors

### 1. Malicious Workflow Author
- **Capability**: Can craft malicious input values in workflow files
- **Motivation**: Code execution, data exfiltration, supply chain compromise
- **Likelihood**: Medium (requires repo write access)

### 2. Malicious Pull Request Author
- **Capability**: Can submit PRs with modified workflow files
- **Motivation**: Inject malicious code into release process
- **Likelihood**: Medium (PRs from forks have limited token scope)

### 3. Compromised Dependency
- **Capability**: Malicious code in go-github, go-yaml, or helm-docs
- **Motivation**: Supply chain attack
- **Likelihood**: Low (but high impact)

### 4. Insider Threat
- **Capability**: Legitimate access but malicious intent
- **Motivation**: Various
- **Likelihood**: Low

## Threats and Mitigations

### T1: Shell Injection via Action Inputs
| | |
|---|---|
| **STRIDE Category** | Tampering, Elevation of Privilege |
| **Attack Vector** | Crafted `bump_type` or other inputs with shell metacharacters |
| **Impact** | Arbitrary command execution on runner |
| **Likelihood** | High (without mitigation) |
| **Mitigation** | ✅ All inputs passed via environment variables, not shell interpolation |
| **Mitigation** | ✅ `bump_type` validated against allowlist (major\|minor\|patch) |
| **Residual Risk** | Low |

### T2: Command Injection via helm-docs Arguments
| | |
|---|---|
| **STRIDE Category** | Tampering, Elevation of Privilege |
| **Attack Vector** | Malicious flags in `helm_docs_args` (e.g., `--execute=malicious.sh`) |
| **Impact** | Arbitrary command execution |
| **Likelihood** | High (without mitigation) |
| **Mitigation** | ✅ Strict allowlist of permitted helm-docs flags |
| **Residual Risk** | Low |

### T3: Path Traversal via File Paths
| | |
|---|---|
| **STRIDE Category** | Information Disclosure, Tampering |
| **Attack Vector** | `version_file` or `version_files[].file` containing `../` sequences |
| **Impact** | Read/write files outside repository root |
| **Likelihood** | Medium (without mitigation) |
| **Mitigation** | ✅ `ValidatePath()` function prevents traversal outside working directory |
| **Residual Risk** | Low |

### T4: YAML Injection
| | |
|---|---|
| **STRIDE Category** | Tampering |
| **Attack Vector** | Malicious YAML path expressions in `version_files[].path` |
| **Impact** | Modify unintended YAML values |
| **Likelihood** | Low |
| **Mitigation** | ✅ Path is used for lookup only, value replacement is surgical |
| **Residual Risk** | Low |

### T5: Token Exposure
| | |
|---|---|
| **STRIDE Category** | Information Disclosure |
| **Attack Vector** | Token logged, included in error messages, or exposed via outputs |
| **Impact** | Unauthorized repository access |
| **Likelihood** | Medium (without mitigation) |
| **Mitigation** | ⚠️ Token passed via environment variable, not logged |
| **Mitigation** | ⚠️ GitHub automatically masks tokens in logs |
| **Residual Risk** | Medium - depends on token scope |

### T6: Denial of Service via Large Files
| | |
|---|---|
| **STRIDE Category** | Denial of Service |
| **Attack Vector** | Extremely large VERSION or YAML files |
| **Impact** | Runner resource exhaustion |
| **Likelihood** | Low |
| **Mitigation** | ⚠️ No explicit file size limits |
| **Residual Risk** | Low (GitHub runner limits provide implicit protection) |

### T7: Supply Chain - Dependency Compromise
| | |
|---|---|
| **STRIDE Category** | Tampering, Elevation of Privilege |
| **Attack Vector** | Malicious code in dependencies (go-github, go-yaml, helm-docs) |
| **Impact** | Arbitrary code execution |
| **Likelihood** | Low |
| **Mitigation** | ⚠️ Use go.sum for dependency verification |
| **Mitigation** | ⚠️ Pin helm-docs version in workflows |
| **Residual Risk** | Medium |

### T8: Git Branch/Tag Manipulation
| | |
|---|---|
| **STRIDE Category** | Tampering, Repudiation |
| **Attack Vector** | Creating release branches that conflict or overwrite existing ones |
| **Impact** | Release integrity compromise |
| **Likelihood** | Low |
| **Mitigation** | ⚠️ Branch names are deterministic (`release/vX.Y.Z`) |
| **Residual Risk** | Low (branch protection rules should be used) |

## Security Controls Summary

### Implemented Controls ✅

| Control | Description | Threats Mitigated |
|---------|-------------|-------------------|
| Environment variable inputs | Inputs passed via env vars, not shell interpolation | T1 |
| Input validation | `bump_type` validated against allowlist | T1 |
| Helm-docs argument allowlist | Only permitted flags accepted | T2 |
| Path validation | `ValidatePath()` prevents directory traversal | T3 |
| Surgical YAML updates | Values replaced precisely, structure preserved | T4 |

### Recommended Additional Controls ⚠️

| Control | Description | Threats Mitigated | Priority |
|---------|-------------|-------------------|----------|
| Token scope documentation | Document minimum required permissions | T5 | High |
| Dependency scanning | Enable Dependabot/Renovate | T7 | Medium |
| SBOM generation | Generate and publish SBOM | T7 | Medium |
| File size limits | Add explicit limits for processed files | T6 | Low |
| Signed releases | Sign release artifacts | T7, T8 | Medium |

## Minimum Token Permissions

The action requires the following GitHub token permissions:

```yaml
permissions:
  contents: write    # Create branches and commits
  pull-requests: write  # Create pull requests
  issues: write      # Add labels to PRs (optional)
```

**Recommendation**: Use a fine-grained personal access token (PAT) or GitHub App token with minimum required permissions rather than `GITHUB_TOKEN` when possible.

## Security Checklist for Users

- [ ] Use a token with minimum required permissions
- [ ] Enable branch protection on `main` branch
- [ ] Require PR reviews before merge
- [ ] Enable required status checks
- [ ] Review the allowlist of helm-docs flags if using that feature
- [ ] Pin the action version (e.g., `@v1.0.11`) rather than using `@main`
- [ ] Enable Dependabot for security updates

## Incident Response

If you discover a security vulnerability in this action:

1. **Do not** open a public issue
2. Email security@stacklok.com with details
3. Include steps to reproduce if possible
4. Allow 90 days for a fix before public disclosure

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025-01-15 | Initial threat model |
