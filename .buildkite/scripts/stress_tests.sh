#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Stress Tests for $beats_subfilder"
make STRESS_TEST_OPTIONS='-timeout=20m -race -v -parallel 1' -C $beats_subfilder stress-tests
