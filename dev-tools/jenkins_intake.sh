#!/usr/bin/env bash
set -euox pipefail

: "${HOME:?Need to set HOME to a non-empty value.}"
: "${WORKSPACE:?Need to set WORKSPACE to a non-empty value.}"

source ./dev-tools/common.bash

jenkins_setup

cleanup() {
  echo "Running cleanup..."
  rm -rf $TEMP_PYTHON_ENV
  echo "Cleanup complete."
}
trap cleanup EXIT

make check
