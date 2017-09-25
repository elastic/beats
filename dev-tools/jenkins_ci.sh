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

# Setup Go.
export GOPATH=${WORKSPACE}
export PATH=${GOPATH}/bin:${PATH}
if [ -f ".go-version" ]; then
  eval "$(gvm $(cat .go-version))"
else
  eval "$(gvm 1.7.5)"
fi

# Workaround for Python virtualenv path being too long.
TEMP_PYTHON_ENV=$(mktemp -d)
export PYTHON_ENV="${TEMP_PYTHON_ENV}/python-env"

cleanup() {
  echo "Running cleanup..."
  rm -rf $TEMP_PYTHON_ENV
  make stop-environment fix-permissions
  echo "Killing all running containers..."
  docker ps -q | xargs -r docker kill || true
  echo "Cleaning stopped docker containers and dangling images/networks/volumes..."
  docker system prune -f || true
  echo "Cleanup complete."
}
trap cleanup EXIT

rm -rf ${GOPATH}/pkg
cd ${beat}
RACE_DETECTOR=1 make clean check testsuite