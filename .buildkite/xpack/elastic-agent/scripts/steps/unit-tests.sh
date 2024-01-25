#!/usr/bin/env bash
set -uo pipefail

source .buildkite/scripts/common.sh

echo "--- Unit tests"
RACE_DETECTOR=true TEST_COVERAGE=true mage unitTest
TESTS_EXIT_STATUS=$?
# Copy coverage file to build directory so it can be downloaded as an artifact
cp build/TEST-go-unit.cov coverage.out
exit $TESTS_EXIT_STATUS