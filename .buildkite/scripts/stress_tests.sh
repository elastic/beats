#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfolder=$1

sudo chmod -R go-w ${beats_subfolder}/

echo "--- Run Stress Tests for $beats_subfolder"
pushd "${beats_subfolder}" > /dev/null

umask 0022
make STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' GOTEST_OUTPUT_OPTIONS='| go-junit-report > libbeat-stress-test.xml' stress-tests

popd > /dev/null
