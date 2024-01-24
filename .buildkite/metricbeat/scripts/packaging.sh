#!/usr/bin/env bash

set -euo pipefail

source .buildkite/metricbeat/scripts/common.sh

echo "--- Evaluate Metricbeat Changes"

changeset=(
  "^metricbeat/.*"
  "^go.mod"
  "^pytest.ini"
  "^dev-tools/.*"
  "^libbeat/.*"
  "^testing/.*"
  )

are_paths_changed() {
  local inputs=("${@}")
  local changelist=()
  for change in "${inputs[@]}"; do
    changed_files=$(git diff --name-only HEAD@{1} HEAD | grep -E "$change")
    if [ -n "$changed_files" ]; then
      changelist+=("${changed_files}")
    fi
  done
  if [[ -n "${changelist[@]}" ]]; then
    echo "Files changed:"
    echo "${changelist[*]}"
    return 0
  else
    local message="No files changed within Metricbeat changeset"
    echo "$message"
    buildkite-agent annotate "$message" --style "info" --context "$BUILDKITE_STEP_KEY"
    return 1
  fi
}

if are_paths_changed "${changeset[@]}" && [ "$BUILDKITE_TAG" == "" ] && [ "$BUILDKITE_PULL_REQUEST" != "" ]; then
  cat <<- YAML | buildkite-agent pipeline upload

env:
  IMAGE_UBUNTU_X86_64: "family/core-ubuntu-2204"
  IMAGE_UBUNTU_ARM_64: "core-ubuntu-2004-aarch64"

steps:

  - group: "Packaging"
    key: "packaging"
    steps:
      - label: ":linux: Packaging Linux"
        key: "packaging-linux"
        command: "pushd "metricbeat" > /dev/null && mage package"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "c2-standard-16"
        env:
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"

      - label: ":linux: Packaging ARM"
        key: "packaging-arm"
        command: "pushd "metricbeat" > /dev/null && mage package"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "t4g.xlarge"
        env:
          PLATFORMS: "linux/arm64"
          PACKAGES: "docker"

YAML

else
  echo "Nothing has changed or it's not a pull request. Skipping packaging..."
  exit 0
fi
