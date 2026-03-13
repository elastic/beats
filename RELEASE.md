# Beats Release Automation

Quick reference guide for Beats release automation using mage.

## Prerequisites

- Go 1.25+
- Git
- GitHub token with repo permissions
- Python with beats-changelog package (for changelog workflows)

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CURRENT_RELEASE` | Yes | - | Version to release (e.g., "9.3.0") |
| `LATEST_RELEASE` | No | - | Previous release version |
| `GITHUB_TOKEN` | Yes* | - | GitHub API token (*not required in DRY_RUN mode) |
| `BASE_BRANCH` | No | "main" | Base branch for PRs |
| `RELEASE_BRANCH` | No | Auto-derived | Release branch name (e.g., "9.3") |
| `PROJECT_OWNER` | No | "elastic" | GitHub repository owner |
| `PROJECT_REPO` | No | "beats" | GitHub repository name |
| `PROJECT_REVIEWERS` | No | "elastic/elastic-agent-release" | Comma-separated reviewers |
| `DRY_RUN` | No | "false" | Set to "true" for testing without push/PR |
| `GIT_AUTHOR_NAME` | No | "github-actions[bot]" | Git commit author name |
| `GIT_AUTHOR_EMAIL` | No | "github-actions[bot]@users.noreply.github.com" | Git commit author email |
| `CHANGELOG_TO_COMMIT` | No | "HEAD" | Commit to generate changelog to |

## Quick Start

### Major/Minor Release (9.3.0)

Creates 1 PR with all version updates:

```bash
export CURRENT_RELEASE="9.3.0"
export GITHUB_TOKEN="ghp_your_token"
export DRY_RUN=true

# Test first (no push/PR)
mage release:runMajorMinor

# Review changes
git status
git diff

# Run for real
export DRY_RUN=false
mage release:runMajorMinor
```

### Patch Release (9.2.1)

Creates 2 PRs (docs+version, test-env):

```bash
git checkout 9.2
git pull

export CURRENT_RELEASE="9.2.1"
export LATEST_RELEASE="9.2.0"
export BASE_BRANCH="9.2"
export GITHUB_TOKEN="ghp_your_token"

mage release:runPatch
```

### Changelog Workflow

Generates changelog and creates 1 PR:

```bash
export CURRENT_RELEASE="9.3.0"
export LATEST_RELEASE="9.2.0"
export RELEASE_BRANCH="9.3"
export GITHUB_TOKEN="ghp_your_token"

mage release:runChangelog
```

## Available Commands

### Workflow Commands (Recommended)

These orchestrate the complete workflow:

```bash
mage release:runMajorMinor    # Major/minor release (1 PR)
mage release:runPatch          # Patch release (2 PRs)
mage release:runChangelog      # Changelog workflow (1 PR)
```

### Individual File Update Commands

Use these for manual updates or testing:

```bash
mage release:updateVersion 9.3.0
mage release:updateDocs 9.3.0
mage release:updateTestEnv 9.2.0 9.3.0
mage release:updateMergify 9.3
```

## DRY_RUN Mode

Always test with `DRY_RUN=true` first:

```bash
export DRY_RUN=true
mage release:runMajorMinor

# Review all changes
git status
git diff

# When satisfied, run for real
export DRY_RUN=false
mage release:runMajorMinor
```

In DRY_RUN mode:
- Files are updated locally
- Branches are created locally
- Commits are made locally
- NO push to remote
- NO PR creation

## Files Updated by Release Automation

| File | Update Type |
|------|-------------|
| `libbeat/version/version.go` | Version constant |
| `libbeat/docs/version.asciidoc` | Stack version, doc branch |
| `deploy/kubernetes/metricbeat-kubernetes.yaml` | Docker image tag |
| `deploy/kubernetes/filebeat-kubernetes.yaml` | Docker image tag |
| `deploy/kubernetes/auditbeat-kubernetes.yaml` | Docker image tag |
| `README.md` | Branch references |
| `testing/environments/docker/elasticsearch_kerberos/Dockerfile` | Docker image tag |
| `testing/environments/latest.yml` | Docker image tag |
| `x-pack/metricbeat/docker-compose.yml` | Docker image tag |
| `metricbeat/module/logstash/docker-compose.yml` | Docker image tag |
| `metricbeat/docker-compose.yml` | Docker image tag |
| `.mergify.yml` | Backport configuration |
| `CHANGELOG.asciidoc` | Changelog entries |

## Troubleshooting

### "working directory is not clean"

Commit or stash your changes:
```bash
git status
git stash
```

### "CURRENT_RELEASE environment variable is required"

Set the required environment variables:
```bash
export CURRENT_RELEASE="9.3.0"
```

### "GITHUB_TOKEN is required when not in dry-run mode"

Set your GitHub token:
```bash
export GITHUB_TOKEN="ghp_your_token"
```

Or use DRY_RUN mode:
```bash
export DRY_RUN=true
```

### "minor releases for version X.x are deprecated and blocked"

Versions 6.x, 7.x, and 8.x minor releases are blocked. Only patch releases are allowed for these versions.

### "failed to push"

Check your git credentials and remote configuration:
```bash
git remote -v
git fetch origin
```

## For More Information

See [dev-tools/mage/release/README.md](dev-tools/mage/release/README.md) for comprehensive documentation.
