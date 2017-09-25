#!/usr/bin/env bash
set -euox pipefail

: "${HOME:?Need to set HOME to a non-empty value.}"
: "${WORKSPACE:?Need to set WORKSPACE to a non-empty value.}"

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
  echo "Cleanup complete."
}
trap cleanup EXIT

make check