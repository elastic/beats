# Overview

This is how we manage dependency updates using `updatecli`.

## Bump VM Images

This directory contains updatecli configuration to automatically update VM image versions in Buildkite pipeline files across the beats repository.

### Overview

The automation tracks the latest VM image builds from the Google Cloud Storage artifacts API and updates corresponding image references in Buildkite pipeline configurations:

- **Source**: `https://storage.googleapis.com/artifacts-api/vm-images/beats/latest.json`
- **Target Pattern**: `platform-ingest-beats-{OS}-{VERSION}` → `platform-ingest-beats-{OS}-{LATEST_VERSION}`

### Files Updated

The automation updates 18 Buildkite pipeline files across the beats repository:

**Root Level Pipelines:**
- `.buildkite/aws-tests-pipeline.yml`
- `.buildkite/packaging.pipeline.yml`

**Beat-Specific Pipelines:**
- `.buildkite/auditbeat/auditbeat-pipeline.yml`
- `.buildkite/filebeat/filebeat-pipeline.yml`
- `.buildkite/heartbeat/heartbeat-pipeline.yml`
- `.buildkite/libbeat/pipeline.libbeat.yml`
- `.buildkite/metricbeat/pipeline.yml`
- `.buildkite/packetbeat/pipeline.packetbeat.yml`
- `.buildkite/winlogbeat/pipeline.winlogbeat.yml`

**X-Pack Pipelines:**
- `.buildkite/x-pack/pipeline.xpack.auditbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.filebeat.yml`
- `.buildkite/x-pack/pipeline.xpack.heartbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.libbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.metricbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.osquerybeat.yml`
- `.buildkite/x-pack/pipeline.xpack.otel.yml`
- `.buildkite/x-pack/pipeline.xpack.packetbeat.yml`
- `.buildkite/x-pack/pipeline.xpack.winlogbeat.yml`

**Deploy Pipelines:**
- `.buildkite/deploy/kubernetes/deploy-k8s-pipeline.yml`

Specifically, it updates the `IMAGE_*` environment variables with the latest VM image versions:

```yaml
env:
  IMAGE_UBUNTU_X86_64: "platform-ingest-beats-ubuntu-2204-YYYYMMDD"
  IMAGE_MACOS_X86_64: "platform-ingest-beats-macos-12-YYYYMMDD"
  # ... other IMAGE_* variables
```

### Configuration Files

- **`updatecli-bump-vm-images.yml`**: Main updatecli configuration that defines sources and targets

### How It Works

The automation:
1. Fetches the latest VM image date from the Google Cloud Storage JSON endpoint
2. Checks if the version differs from what's currently in `.buildkite/pipeline.yml`
3. If a new version is detected, updates all 24 pipeline files using a single regex pattern match
4. Creates a pull request with the changes

### Pull Request Details

When new versions are detected, the automation creates a pull request with:
- **Title**: `[{BRANCH_NAME}][Automation] Bump VM Image version to {LATEST_VERSION}`
- **Labels**: `dependencies`, `backport-skip`, `skip-changelog`
- **Auto-merge**: Disabled (requires manual review)

### Pattern Matching

The configuration uses a regex pattern to match and replace VM image references:
- **Match Pattern**: `(IMAGE_.+): "platform-ingest-beats-(.+)-(.+)"`
- **Replace Pattern**: `$1: "platform-ingest-beats-$2-{LATEST_VERSION}"`

This preserves the image type (OS/architecture) while updating only the date/version component.

### Manual Testing

To test the configuration locally:

```bash
export GITHUB_TOKEN=$(gh auth token)
export GITHUB_ACTOR=your-github-username
export BRANCH_NAME=main
updatecli diff \
    --config .github/workflows/updatecli.d/updatecli-bump-vm-images.yml \
    --values .github/workflows/updatecli.d/values.d/scm.yml

# Apply changes (requires write access to beats repo)
updatecli apply \
    --config .github/workflows/updatecli.d/updatecli-bump-vm-images.yml \
    --values .github/workflows/updatecli.d/values.d/scm.yml
```