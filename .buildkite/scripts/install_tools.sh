#!/bin/bash

source .buildkite/scripts/common.sh

set -euox pipefail

echo "--- Env preparation"

DEBIAN_FRONTEND="noninteractive"
sudo mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null

add_bin_path

if command -v docker-compose &> /dev/null
then
  echo "Found docker-compose. Checking version.."
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version | awk '{print $3}'|sed s/\,//)
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  else
    echo "Versions don't match. Need to install the correct version of docker-compose."
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
else
  with_docker_compose "${DOCKER_COMPOSE_VERSION}"
fi

with_go "${GO_VERSION}"
with_mage
with_python
