#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

echo "--- Env preparation"

# Temporary solution to fix the issues with "sudo apt get...." https://elastic.slack.com/archives/C0522G6FBNE/p1706003603442859?thread_ts=1706003209.424539&cid=C0522G6FBNE
# It could be removed when we use our own image for the BK agent.
if [ "${platform_type}" == "Linux" ]; then
  if [ "${platform_type}" == "Linux" ]; then
    if [ $(checkLinuxType) = "ubuntu" ]; then
      DEBIAN_FRONTEND="noninteractive"
      #sudo command doesn't work at the "pre-command" hook because of another user environment (root with strange permissions)
      sudo mkdir -p /etc/needrestart
      echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null
    fi
  fi
fi

add_bin_path

if command -v docker-compose &> /dev/null
then
  echo "Found docker-compose. Checking version.."
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}'|sed s/\,//)
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  elif [[ "${platform_type}" == "Linux" && "${arch_type}" == "aarch64" ]]; then
    with_docker_compose "${DOCKER_COMPOSE_VERSION_AARCH64}"
  elif [[ "${platform_type}" == "Linux" && "${arch_type}" == "x86_64" ]]; then
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
else
  with_docker_compose "${DOCKER_COMPOSE_VERSION}"
fi

with_go "${GO_VERSION}"
with_mage
with_python
with_dependencies
config_git
mage dumpVariables

#sudo command doesn't work at the "pre-command" hook because of another user environment (root with strange permissions)
sudo chmod -R go-w "${BEATS_PROJECT_NAME}/"     #TODO: Remove when the issue is solved https://github.com/elastic/beats/issues/37838

pushd "${BEATS_PROJECT_NAME}" > /dev/null

#TODO "umask 0022" has to be removed after our own image is ready (it has to be moved to the image)
umask 0022    # fix the filesystem permissions issue like this: https://buildkite.com/elastic/beats-metricbeat/builds/1329#018d3179-25a9-475b-a2c8-64329dfe092b/320-1696

popd > /dev/null
