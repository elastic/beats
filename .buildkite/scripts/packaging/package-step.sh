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
      - label: ":ubuntu: ${BEATS_PROJECT_NAME}/Packaging Linux X86"
        key: "package-linux-x86"
        env:
<<<<<<< HEAD:.buildkite/auditbeat/scripts/package-step.sh
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64"
        command:
          - ".buildkite/auditbeat/scripts/package.sh"
=======
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
          SNAPSHOT: true
        command: ".buildkite/scripts/packaging/package.sh"
>>>>>>> 01f5a90209 (Snapshot packaging stage migration (#37827)):.buildkite/scripts/packaging/package-step.sh
        notify:
          - github_commit_status:
              context: "${BEATS_PROJECT_NAME}/Packaging: Linux X86"
        agents:
          provider: gcp
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"

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
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
YAML
)
  echo "${bk_pipeline}" | buildkite-agent pipeline upload
else
  buildkite-agent annotate "No required files changed. Skipped packaging" --style 'warning' --context 'ctx-warning'
  exit 0
fi
