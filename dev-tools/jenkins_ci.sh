#!/usr/bin/env bash
set -euox pipefail

: "${HOME:?Need to set HOME to a non-empty value.}"
: "${WORKSPACE:?Need to set WORKSPACE to a non-empty value.}"
: "${beat:?Need to set beat to a non-empty value.}"

if [ ! -d "$beat" ]; then
  echo "$beat does not exist"
  mkdir -p build
  touch build/TEST-empty.xml
  exit
fi

source ./dev-tools/common.bash

jenkins_setup

cleanup() {
  echo "Running cleanup..."
  rm -rf $TEMP_PYTHON_ENV

  if docker info > /dev/null ; then
    make stop-environment || true
    make fix-permissions || true
    echo "Killing all running containers..."
    ids=$(docker ps -q)
    if [ -n "$ids" ]; then
      docker kill $ids
    fi  
    echo "Cleaning stopped docker containers and dangling images/networks/volumes..."
    docker system prune -f || true
  fi
  echo "Cleanup complete."
}
trap cleanup EXIT

rm -rf ${GOPATH}/pkg
cd ${beat}
RACE_DETECTOR=1 make clean check testsuite
