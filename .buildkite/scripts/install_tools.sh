#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

echo "--- Env preparation"

if [ "${platform_type}" == "Linux" ]; then
  DEBIAN_FRONTEND="noninteractive"
  sudo mkdir -p /etc/needrestart
  echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null
fi

add_bin_path

if command -v docker-compose &> /dev/null
then
  echo "Found docker-compose. Checking version.."
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version | awk '{print $4}'|sed s/\,//)
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  elif [ "${platform_type}" == "Linux" && "${arch_type}" == "aarch64" ]; then
    with_docker_compose "${DOCKER_COMPOSE_VERSION_AARCH64}"
  elif [ "${platform_type}" == "Linux" && "${arch_type}" == "x86_64" ]; then
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
else
  with_docker_compose "${DOCKER_COMPOSE_VERSION}"
fi

with_go "${GO_VERSION}"
with_mage
with_python
