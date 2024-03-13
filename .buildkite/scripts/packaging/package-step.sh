#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

changeset="^${BEATS_PROJECT_NAME}/
^go.mod
^pytest.ini
^dev-tools/
^libbeat/
^testing/
^\.buildkite/${BEATS_PROJECT_NAME}/"

if are_files_changed "$changeset"; then
  bk_pipeline=$(cat <<-YAML
    steps:
      - label: ":linux: ${BEATS_PROJECT_NAME}/Packaging Linux ARM"
        key: "package-linux-arm"
        env:
          PLATFORMS: "linux/arm64"
          PACKAGES: "docker"
          SNAPSHOT: true
        command: ".buildkite/scripts/packaging/package.sh"
        notify:
          - github_commit_status:
              context: "${BEATS_PROJECT_NAME}/Packaging: ARM"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.xlarge"
YAML
)
  echo "${bk_pipeline}" | buildkite-agent pipeline upload
else
  buildkite-agent annotate "No required files changed. Skipped packaging" --style 'warning' --context 'ctx-warning'
  exit 0
fi
