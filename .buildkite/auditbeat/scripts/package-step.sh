#!/usr/bin/env bash

set -euo pipefail

changeset="^auditbeat/
    ^go.mod
    ^pytest.ini
    ^dev-tools/
    ^libbeat/
    ^testing/
    ^\.buildkite/auditbeat/"

if are_files_changed "$changeset"; then
  cat <<-EOF
    steps:
      - label: ":ubuntu: Packaging Linux X86"
        key: "package-linux-x86"
        command:
          - ".buildkite/auditbeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "auditbeat/Packaging: Linux X86"
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
              context: "auditbeat/Packaging: ARM"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.large"
EOF
fi

