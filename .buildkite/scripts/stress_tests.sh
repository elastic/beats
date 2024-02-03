#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_subfolder=$1

echo "--- Run Stress Tests for $beats_subfolder"
pushd "${beats_subfolder}" > /dev/null

make STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' GOTEST_OUTPUT_OPTIONS='| go-junit-report > libbeat-stress-test.xml' stress-tests

popd > /dev/null
