#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfolder=$1

echo "--- Run Stress Tests for $beats_subfolder"
make STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C $beats_subfolder stress-tests
