#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

changeset="^auditbeat/
^go.mod
^pytest.ini
^dev-tools/
^libbeat/
^testing/
^\.buildkite/auditbeat/"

if are_files_changed "$changeset"; then
  bk_pipeline=$(cat <<-YAML
    steps:
      - label: ":ubuntu: Packaging Linux X86"
        key: "package-linux-x86"
        env:
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64"
        command:
          - ".buildkite/auditbeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "Auditbeat/Packaging: Linux X86"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"

      - label: ":linux: Packaging Linux ARM"
        key: "package-linux-arm"
        env:
          PLATFORMS: "linux/arm64"
          PACKAGES: "docker"
        command:
          - ".buildkite/auditbeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "Auditbeat/Packaging: ARM"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.large"
YAML
)
  echo "${bk_pipeline}" | buildkite-agent pipeline upload
else
  buildkite-agent annotate "No required files changed. Skipped packaging" --style 'warning' --context 'ctx-warning'
  exit 0
fi
