#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_project=$1

echo "--- Run Packaging for $beats_project"
pushd "${beats_project}" > /dev/null

mage package

popd > /dev/null

# if are_paths_changed "${metricbeat_changeset[@]}" ; then
#   cat <<- YAML | buildkite-agent pipeline upload

# env:
#   IMAGE_UBUNTU_X86_64: "family/core-ubuntu-2204"
#   IMAGE_UBUNTU_ARM_64: "core-ubuntu-2004-aarch64"

# steps:

#   - group: "Packaging"
#     key: "packaging"
#     steps:
#       - label: ":linux: Packaging Linux"
#         key: "packaging-linux"
#         command: "pushd "metricbeat" > /dev/null && mage package"
#         agents:
#           provider: "gcp"
#           image: "${IMAGE_UBUNTU_X86_64}"
#           machineType: "c2-standard-16"
#         env:
#           PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"

#       - label: ":linux: Packaging ARM"
#         key: "packaging-arm"
#         command: "pushd "metricbeat" > /dev/null && mage package"
#         agents:
#           provider: "aws"
#           imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
#           instanceType: "t4g.xlarge"
#         env:
#           PLATFORMS: "linux/arm64"
#           PACKAGES: "docker"

# YAML

# else
#   echo "Nothing has changed or it's not a pull request. Skipping packaging..."
#   exit 0
# fi
