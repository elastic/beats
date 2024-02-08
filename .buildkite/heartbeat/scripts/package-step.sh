#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

changeset="^heartbeat/
^go.mod
^pytest.ini
^dev-tools/
^libbeat/
^testing/
^\.buildkite/heartbeat/"

if are_files_changed "$changeset"; then
  cat <<-EOF
    steps:
      - label: ":ubuntu: Packaging Linux X86"
        key: "package-linux-x86"
        env:
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
        command:
          - ".buildkite/heartbeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "heartbeat/Packaging: Linux X86"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"

      - label: ":linux: Packaging Linux ARM"
        key: "package-linux-arm"
        env:
          PLATFORMS: "linux/arm64"
          PACKAGES: "docker"
        command:
          - ".buildkite/heartbeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "heartbeat/Packaging: ARM"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.large"
EOF
else
  cat <<-EOF
    steps:
      - label: "Skipping packaging"
        key: "package-skip"
        command:
          - "buildkite-agent annotate "No required files changed." --style 'warning' --context 'ctx-warning'"
        notify:
          - github_commit_status:
              context: "heartbeat/package-skip"
EOF
fi
