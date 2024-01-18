#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

# echo "--- prepare env"
# add_bin_path
# with_go ${GO_VERSION}
# with_mage

echo "--- run intergration tests"
pushd "metricbeat" > /dev/null

mage goIntegTest

popd > /dev/null
