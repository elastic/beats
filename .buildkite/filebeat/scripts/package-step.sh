#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

changeset="^filebeat/
^go.mod
^pytest.ini
^dev-tools/
^libbeat/
^testing/
^\.buildkite/filebeat/"

if are_files_changed "$changeset"; then
  bk_pipeline=$(cat <<-YAML
    steps:
      - label: ":ubuntu: Packaging Linux X86"
        key: "package-linux-x86"
        env:
<<<<<<< HEAD:.buildkite/filebeat/scripts/package-step.sh
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
        command:
          - ".buildkite/filebeat/scripts/package.sh"
=======
          PLATFORMS: $PACKAGING_PLATFORMS
          SNAPSHOT: true
        command: ".buildkite/scripts/packaging/package.sh"
>>>>>>> 80dab50f0c (replace default images (#38583)):.buildkite/scripts/packaging/package-step.sh
        notify:
          - github_commit_status:
              context: "Filebeat/Packaging: Linux X86"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"

      - label: ":linux: Packaging Linux ARM"
        key: "package-linux-arm"
        env:
          PLATFORMS: $PACKAGING_ARM_PLATFORMS
          PACKAGES: "docker"
        command:
          - ".buildkite/filebeat/scripts/package.sh"
        notify:
          - github_commit_status:
              context: "Filebeat/Packaging: ARM"
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
