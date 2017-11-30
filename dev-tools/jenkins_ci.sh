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

# Cleanup before starting build in case some previous build was canceled and did not fully clean up
cleanup

trap cleanup EXIT

rm -rf ${GOPATH}/pkg
cd ${beat}
RACE_DETECTOR=1 make clean check testsuite
