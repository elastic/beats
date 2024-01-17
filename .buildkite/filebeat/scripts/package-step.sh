#!/usr/bin/env bash

set -euo pipefail

source .buildkite/filebeat/scripts/common.sh

changeset="^filebeat/
    ^go.mod
    ^pytest.ini
    ^dev-tools/
    ^libbeat/
    ^testing/
    ^\.buildkite/filebeat/"

if are_files_changed "$changeset"; then
  cat <<-EOF
  - group: "Packaging"
    key: "packaging"
    if: build.env("BUILDKITE_PULL_REQUEST") != "false"
    depends_on:
      - "mandatory-tests"
      - "extended-tests"
      - "extended-tests-win"

    steps:
      - label: ":ubuntu: Packaging Linux X86"
        key: "package-linux-x86"
        command:
          - "sudo apt-get update"
          - "sudo apt-get install -y python3-venv"
          - ".buildkite/filebeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "Filebeat/Packaging: Linux X86"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"

      - label: ":linux: Packaging Linux ARM"
        key: "package-linux-arm"
        env:
          PLATFORMS: "linux/arm64"
          PACKAGES: "docker"
        command:
          - "sudo apt-get update"
          - "sudo apt-get install -y python3-venv"
          - ".buildkite/filebeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "Filebeat/Packaging: ARM"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.large"
EOF
fi
