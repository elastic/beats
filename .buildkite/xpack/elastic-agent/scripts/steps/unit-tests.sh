#!/usr/bin/env bash
set -uo pipefail
SOURCE_DIR="xpack/elastic-agent"
PIPELINE_DIR=".buildkite/xpack/elastic-agent"

source "$PIPELINE_DIR/scripts/steps/common.sh"

echo "--- Unit tests"
RACE_DETECTOR=true TEST_COVERAGE=true mage -d "$SOURCE_DIR" build unitTest
TESTS_EXIT_STATUS=$?

exit $TESTS_EXIT_STATUS