#!/usr/bin/env bash
set -euox pipefail

: "${HOME:?Need to set HOME to a non-empty value.}"
: "${WORKSPACE:?Need to set WORKSPACE to a non-empty value.}"

source $(dirname "$0")/common.bash

jenkins_setup
docker_setup

cleanup() {
  echo "Running cleanup..."
  rm -rf $TEMP_PYTHON_ENV

  if docker info > /dev/null ; then
    echo "Killing all running containers..."
    ids=$(docker ps -q)
    if [ -n "$ids" ]; then
      docker kill $ids
    fi  
    echo "Cleaning stopped docker containers and dangling images/networks/volumes..."
    docker system prune -f || true

    os=$(uname -s)
    if [ "$os" == "Darwin" ]; then
      # Clean images accept for the ones we're currently using in order to
      # gain some disk space.
      echo "Disk space before image cleanup:"
      df -h /
      docker system df
      echo "Cleaning images"
      docker images --format "{{.ID}} {{.Repository}}:{{.Tag}}" \
        | grep -v "docker.elastic.co/beats-dev/golang-crossbuild:$(cat .go-version)-" \
        | awk '{print $1}' \
        | xargs docker rmi -f || true
      echo "Disk space after image cleanup:"
      df -h /
      docker system df
    fi
  fi

  echo "Cleanup complete."
}
trap cleanup EXIT

# This controls the defaults used the Jenkins package job. They can be
# overridden by setting them in the environement prior to running this script.
export SNAPSHOT="${SNAPSHOT:-true}"
export PLATFORMS="${PLATFORMS:-+linux/armv7 +linux/ppc64le +linux/s390x +linux/mips64}"

make release
