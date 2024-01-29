#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

echo "--- Env preparation"
DEBIAN_FRONTEND="noninteractive"
sudo mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null
add_bin_path
with_go "${GO_VERSION}"
with_mage
with_python

if command -v docker-compose &> /dev/null
then
  set +e
  echo "Found docker-compose. Checking version.."
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version|awk '{print $3}'|sed s/\,//)
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  else
    echo "Versions don't match. Need to install the correct version of docker-compose."
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
  set -e
fi

echo "--- Evaluate Metricbeat Changes"

echo "--- Run Packaging for $beats_subfilder"
pushd "${beats_subfilder}" > /dev/null

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
