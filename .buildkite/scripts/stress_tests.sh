#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Stress Tests for $BEATS_PROJECT_NAME"

pushd "${BEATS_PROJECT_NAME}" > /dev/null

make STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' GOTEST_OUTPUT_OPTIONS='| go-junit-report > libbeat-stress-test.xml' stress-tests

popd > /dev/null
