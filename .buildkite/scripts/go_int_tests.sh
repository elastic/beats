#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Env preparation"

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

echo "--- Run Go Intergration Tests for $beats_subfilder"
pushd "${beats_subfilder}" > /dev/null

mage goIntegTest

popd > /dev/null
