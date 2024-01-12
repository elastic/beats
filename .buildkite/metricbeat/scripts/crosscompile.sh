#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- prepare env"
add_bin_path
with_go ${GO_VERSION}
with_mage

echo "--- run crosscompile"
make -C metricbeat crosscompile
