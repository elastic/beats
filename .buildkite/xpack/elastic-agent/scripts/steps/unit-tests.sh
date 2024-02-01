#!/usr/bin/env bash
set -uo pipefail

source ".buildkite/xpack/elastic-agent/scripts/steps/common.sh"

SOURCE_DIR="$WORKSPACE/xpack/elastic-agent"

echo "--- Unit tests"
RACE_DETECTOR=true TEST_COVERAGE=true mage -d "$SOURCE_DIR" build unitTest
TESTS_EXIT_STATUS=$?

exit $TESTS_EXIT_STATUS