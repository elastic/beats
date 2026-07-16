# Beats Release Automation

Comprehensive guide for Beats release automation using mage.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Release Workflows](#release-workflows)
- [Individual Commands](#individual-commands)
- [DRY_RUN Mode](#dry_run-mode)
- [Multi-PR Workflows](#multi-pr-workflows)
- [Changelog Workflow](#changelog-workflow)
- [FAQ](#faq)
- [Troubleshooting](#troubleshooting)

## Overview

This package provides release automation for the Beats project, migrated from Makefile to mage for better type safety, testing, and maintainability.

**Key features:**
- Pure Go implementation (except changelog - uses Python beats-changelog)
- Comprehensive testing (>60% coverage)
- DRY_RUN mode for safe testing
- Multi-PR workflow support
- Deprecated version checks

**Workflows supported:**
1. **Major/Minor Release (feature-freeze)** - Creates release branch + 4 grouped PRs (backport+version on main, ff-release, docs+test-env on main, next patch on release branch)
2. **Patch Release** - Creates up to 3 PRs (version, docs, test-env)
3. **Changelog** - Generates changelog and creates 1 PR

## Prerequisites

### Required Tools

- **Go** 1.22 or later
- **Git** 2.30 or later
- **GitHub CLI** (optional, for advanced usage)
- **Python** 3.8+ with beats-changelog package (for changelog workflows only)

### GitHub Token

Create a personal access token with `repo` scope:

1. Go to https://github.com/settings/tokens
2. Click "Generate new token (classic)"
3. Select `repo` scope
4. Copy the token

Store it securely:
```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

### Python Setup (Changelog Only)

If you need to run changelog workflows:

```bash
# Install beats-changelog package
pip install ./release_scripts/beats-changelog

# Or use a virtual environment
python -m venv venv
source venv/bin/activate
pip install ./release_scripts/beats-changelog
```

## Installation

The release automation is built into the Beats mage targets. No separate installation needed.

Verify it's available:
```bash
mage -l | grep release
```

You should see:
```
release:runChangelog     Executes the complete changelog workflow
release:runMajorMinor    Feature-freeze workflow (release branch + 4 grouped PRs)
release:runPatch         Executes the complete patch release workflow
release:updateDocs       Updates version references in documentation and K8s manifests
release:updateMergify    Updates .mergify.yml backport configuration
release:updateTestEnv    Updates docker-compose.yml files with new versions
release:updateVersion    Updates the version in libbeat/version/version.go
```

## Configuration

All configuration is done via environment variables.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CURRENT_RELEASE` | Version to release | `"9.3.0"` |
| `GITHUB_TOKEN` | GitHub API token | `"ghp_..."` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LATEST_RELEASE` | Auto from GitHub (or patch−1) | Optional override for previous release used in test-env updates |
| `BASE_BRANCH` | `"main"` | Base branch for PRs |
| `RELEASE_BRANCH` | Auto-derived from `CURRENT_RELEASE` | Release branch name |
| `PROJECT_OWNER` | `"elastic"` | GitHub repository owner |
| `PROJECT_REPO` | `"beats"` | GitHub repository name |
| `PROJECT_REVIEWERS` | `"elastic/elastic-agent-release"` | Comma-separated reviewers |
| `DRY_RUN` | `"false"` | Set to `"true"` for testing |
| `GIT_AUTHOR_NAME` | `"github-actions[bot]"` | Git commit author name |
| `GIT_AUTHOR_EMAIL` | `"github-actions[bot]@users.noreply.github.com"` | Git commit author email |
| `CHANGELOG_TO_COMMIT` | `"HEAD"` | Commit to generate changelog to |

### Example Configuration

```bash
# Minimal configuration
export CURRENT_RELEASE="9.3.0"
export GITHUB_TOKEN="ghp_your_token"

# Full configuration
export CURRENT_RELEASE="9.3.0"
# LATEST_RELEASE is optional; auto-resolved from elastic/beats GitHub releases
export BASE_BRANCH="main"
export RELEASE_BRANCH="9.3"
export GITHUB_TOKEN="ghp_your_token"
export PROJECT_REVIEWERS="elastic/elastic-agent-release,user1,user2"
export DRY_RUN="true"
```

## Release Workflows

### Major/Minor Release (Feature Freeze)

Runs at a new major or minor version (e.g., 9.0.0, 9.5.0). This is the
feature-freeze workflow, equivalent to ingest-dev `prepare-major-minor-release`
with grouped PRs.

**What it does:**
1. Validates version and checks for deprecated releases
2. Creates release branch from `BASE_BRANCH` (e.g., `9.5`)
3. Prepares **4 grouped PRs** (merge order matters):

| Step | PR | Target | Branch | Changes |
|------|-----|--------|--------|---------|
| 1 | PR-A | `main` | `ff-prep-main-X.Y.0` | Mergify backport rule + `version.go` → next minor |
| 2 | PR-B | release branch | `ff-release-X.Y.0` | ff-release version, docs, test env, `make update` |
| 3 | PR-C | `main` | `ff-prep-main-docs-env-X.(Y+1).0` | Docs + test env for next minor |
| 4 | PR-D | release branch | `ff-prep-next-patch-X.Y.1` | Next patch version + test env |

4. Pushes release branch and opens PRs (unless `DRY_RUN`)

**RM merge order:** push branch → merge PR-A on `main` → merge PR-B on release branch → merge PR-C on `main` → merge PR-D on release branch (after release day).

**Usage:**

```bash
# Test first with DRY_RUN
export CURRENT_RELEASE="9.5.0"
export GITHUB_TOKEN="ghp_your_token"
export DRY_RUN=true

mage release:runMajorMinor

# Review branches
git branch | grep -E 'ff-prep|ff-release|9\.5'

# Run for real
export DRY_RUN=false
mage release:runMajorMinor
```

`LatestRelease` is resolved automatically from published `elastic/beats` GitHub releases (highest same-major version strictly less than `CURRENT_RELEASE`, e.g. `9.5.0` → `9.4.3`). Set `LATEST_RELEASE` only to override.

**Blocked versions:**
- 6.x minor releases (6.5.0, etc.)
- 7.x minor releases (7.5.0, etc.)
- 8.x minor releases (8.5.0, etc.)

Only 9.x+ minor releases are allowed. Patch releases are allowed for all versions.

### Patch Release

Creates a patch release (e.g., 9.2.1) on an existing release branch.

**What it does:**
1. Validates version
2. Creates 2 branches:
   - `update-docs-version-X.Y.Z` - for docs and version
   - `update-testing-env-X.Y.Z` - for test environment
3. Makes updates on each branch
4. Commits changes on each branch
5. Pushes both branches (unless DRY_RUN)
6. Creates 2 PRs (unless DRY_RUN):
   - PR #1: Docs and version updates
   - PR #2: Test environment updates

**Usage:**

```bash
# Checkout the release branch first
git checkout 9.2
git pull

# Configure and run
export CURRENT_RELEASE="9.2.1"
export LATEST_RELEASE="9.2.0"
export BASE_BRANCH="9.2"
export GITHUB_TOKEN="ghp_your_token"

mage release:runPatch
```

### Changelog Workflow

Generates changelog entries and creates a PR.

**What it does:**
1. Creates branch `prepare-changelog-X.Y.Z`
2. Runs `beats-changelog` Python tool
3. Generates changelog entries
4. Commits changes
5. Pushes to remote (unless DRY_RUN)
6. Creates 1 PR (unless DRY_RUN)

**Prerequisites:**
- Python with beats-changelog package installed
- `beats-changelog` command available in PATH

**Usage:**

```bash
export CURRENT_RELEASE="9.3.0"
export LATEST_RELEASE="9.2.0"
export RELEASE_BRANCH="9.3"
export GITHUB_TOKEN="ghp_your_token"

mage release:runChangelog
```

## Individual Commands

For manual updates or testing individual steps.

### UpdateVersion

Updates the version constant in `libbeat/version/version.go`:

```bash
mage release:updateVersion 9.3.0
```

**What it does:**
- Reads `libbeat/version/version.go`
- Replaces `const defaultBeatVersion = "X.Y.Z"`
- Writes the file back

### UpdateDocs

Updates version references in documentation and K8s manifests:

```bash
mage release:updateDocs 9.3.0
```

**Files updated:**
- `libbeat/docs/version.asciidoc` - `:stack-version:`, `:doc-branch:`
- `deploy/kubernetes/metricbeat-kubernetes.yaml` - Docker image tag
- `deploy/kubernetes/filebeat-kubernetes.yaml` - Docker image tag
- `deploy/kubernetes/auditbeat-kubernetes.yaml` - Docker image tag
- `README.md` - Branch references

### UpdateTestEnv

Updates Docker image versions in test environment files:

```bash
mage release:updateTestEnv 9.2.0 9.3.0
```

Arguments:
1. `latest` - Previous version to replace
2. `current` - New version to use

**Files updated:**
- `testing/environments/latest.yml`
- `x-pack/metricbeat/docker-compose.yml`
- `metricbeat/module/logstash/docker-compose.yml`
- `metricbeat/docker-compose.yml`

### UpdateMergify

Updates `.mergify.yml` to add a backport rule for the release branch:

```bash
mage release:updateMergify 9.5
```

Idempotent: skips if a `backport-9.5` rule already exists.

## DRY_RUN Mode

Always test with `DRY_RUN=true` before running for real.

### What DRY_RUN Does

**Executes:**
- File updates
- Branch creation (local only)
- Commits (local only)
- All validation checks

**Skips:**
- Push to remote
- PR creation
- GitHub API calls

### Example Workflow

```bash
# Step 1: Test with DRY_RUN
export CURRENT_RELEASE="9.3.0"
export GITHUB_TOKEN="ghp_your_token"  # Still needed for validation
export DRY_RUN=true

mage release:runMajorMinor

# Step 2: Review changes
git status
git diff
git log --oneline

# Step 3: Test on a fork (optional)
git push fork update-version-9.3.0

# Step 4: If satisfied, reset and run for real
git reset --hard origin/main
export DRY_RUN=false
mage release:runMajorMinor
```

### DRY_RUN Output Example

```
=== Starting Major/Minor Release Workflow ===
Creating release branch: 9.5

--- Preparing PR-A: backport rule + version 9.6.0 on main ---
--- Preparing PR-B: ff-release 9.5.0 on 9.5 ---
--- Preparing PR-C: docs + test env 9.6.0 on main ---
--- Preparing PR-D: next patch 9.5.1 on 9.5 ---

DRY RUN: Skipping push and PR creation
Release branch prepared: 9.5
Branch prepared: ff-prep-main-9.5.0
Branch prepared: ff-release-9.5.0
Branch prepared: ff-prep-main-docs-env-9.6.0
Branch prepared: ff-prep-next-patch-9.5.1
```

## Multi-PR Workflows

Some workflows create multiple PRs to separate concerns.

### Patch Release (3 PRs)

**PR #1: Version**
- Branch: `update-version-X.Y.Z`
- Updates: `libbeat/version/version.go`
- Labels: `release`, `Team:Automation`, `skip-changelog`

**PR #2: Docs**
- Branch: `update-docs-X.Y.Z`
- Updates: docs versions, K8s manifests
- Labels: `docs`, `in progress`, `release`, `Team:Automation`, `skip-changelog`

**PR #3: Test Environment**
- Branch: `update-testing-env-X.Y.Z`
- Updates: All docker-compose files
- Labels: `release`, `Team:Automation`, `skip-changelog`

**Why separate PRs?**
- Docs/version updates need review by docs team
- Test env updates can be merged independently
- Allows parallel review and merge

### Reviewing Multi-PR Workflows

```bash
# After running the workflow, you'll see:
=== Patch Release Workflow Complete ===
PR 1: https://github.com/elastic/beats/pull/12345
PR 2: https://github.com/elastic/beats/pull/12346

# Review each PR
gh pr view 12345
gh pr view 12346

# Approve and merge when ready
gh pr review 12345 --approve
gh pr merge 12345
gh pr review 12346 --approve
gh pr merge 12346
```

## Changelog Workflow

The changelog workflow integrates with the Python `beats-changelog` package.

### Prerequisites

Install beats-changelog:

```bash
# Option 1: System-wide
pip install ./release_scripts/beats-changelog

# Option 2: Virtual environment
python -m venv venv
source venv/bin/activate
pip install ./release_scripts/beats-changelog

# Verify installation
which beats-changelog
beats-changelog --help
```

### Running the Workflow

```bash
export CURRENT_RELEASE="9.3.0"
export LATEST_RELEASE="9.2.0"
export RELEASE_BRANCH="9.3"
export GITHUB_TOKEN="ghp_your_token"
export CHANGELOG_TO_COMMIT="HEAD"  # Or specific commit

mage release:runChangelog
```

### What Gets Updated

The workflow runs:
```bash
beats-changelog split --from vX.Y.Z --to <commit>
```

And updates:
- `CHANGELOG.asciidoc`
- `CHANGELOG.next.asciidoc`
- `libbeat/docs/release.asciidoc`

### Manual Changelog Generation

If you need to generate changelog manually:

```bash
cd release_scripts/beats-changelog
beats-changelog split --from v9.2.0 --to HEAD
```

## FAQ

### Can I run these commands on a fork?

Yes! Set the `PROJECT_OWNER` environment variable:

```bash
export PROJECT_OWNER="your-github-username"
export PROJECT_REPO="beats"
mage release:runMajorMinor
```

### What if I need to update only specific files?

Use the individual commands:

```bash
mage release:updateVersion 9.3.0
# Review the change
git diff libbeat/version/version.go
```

### Can I customize PR reviewers?

Yes, use the `PROJECT_REVIEWERS` variable:

```bash
export PROJECT_REVIEWERS="user1,user2,team1"
```

### How do I test without a GitHub token?

Use DRY_RUN mode:

```bash
export DRY_RUN=true
export CURRENT_RELEASE="9.3.0"
# GITHUB_TOKEN not required in DRY_RUN mode
mage release:runMajorMinor
```

### What if a workflow fails partway through?

The workflows are designed to be resumable. Check the error message and:

1. Fix the issue
2. Reset your branch: `git reset --hard origin/main`
3. Run the workflow again

Or continue manually from where it failed.

### Can I run these workflows in CI/CD?

Yes! These are designed for both local and CI/CD use:

```yaml
# GitHub Actions example
- name: Run major/minor release
  env:
    CURRENT_RELEASE: ${{ inputs.version }}
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: mage release:runMajorMinor
```

## Troubleshooting

### Error: "working directory is not clean"

**Cause:** You have uncommitted changes.

**Solution:**
```bash
git status
git stash
# Or commit your changes
git add .
git commit -m "WIP"
```

### Error: "CURRENT_RELEASE environment variable is required"

**Cause:** Missing required environment variable.

**Solution:**
```bash
export CURRENT_RELEASE="9.3.0"
```

### Error: "GITHUB_TOKEN is required when not in dry-run mode"

**Cause:** Running without DRY_RUN but no GitHub token.

**Solution:**
```bash
# Option 1: Set token
export GITHUB_TOKEN="ghp_your_token"

# Option 2: Use DRY_RUN
export DRY_RUN=true
```

### Error: "minor releases for version X.x are deprecated and blocked"

**Cause:** Trying to create a 6.x, 7.x, or 8.x minor release.

**Solution:**
- Only patch releases allowed for these versions
- Use 9.x or later for minor releases

### Error: "beats-changelog not found in PATH"

**Cause:** Python beats-changelog package not installed.

**Solution:**
```bash
pip install ./release_scripts/beats-changelog
which beats-changelog
```

### Error: "failed to push"

**Cause:** Git credentials or remote configuration issue.

**Solution:**
```bash
# Check remote
git remote -v

# Test access
git fetch origin

# Configure credentials
git config --global credential.helper store
```

### Error: "failed to create PR"

**Cause:** GitHub API issue or permissions.

**Solution:**
```bash
# Verify token has 'repo' scope
# Check GitHub API status: https://www.githubstatus.com/

# Test token manually
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/user
```

### PRs created but labels not added

**Cause:** Token may lack permissions to add labels.

**Solution:**
- Ensure token has `repo` scope
- Add labels manually after PR creation

## Development

### Running Tests

```bash
cd dev-tools/mage/release
go test -v
```

### Test Coverage

```bash
go test -cover
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Target coverage:
- Overall: >60%
- Core functions: >80%

### Adding New Workflows

1. Add workflow function in `workflows.go`
2. Add tests in `workflows_test.go`
3. Add mage target in `magefile.go`
4. Update documentation

## Support

For issues or questions:
- File an issue: https://github.com/elastic/beats/issues
- Check existing issues for similar problems
- Include DRY_RUN output in bug reports
