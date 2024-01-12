#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- prepare env"
add_bin_path
with_go ${GO_VERSION}
with_mage
with_python

echo "--- run intergration tests"
pushd "metricbeat" > /dev/null

mage pythonIntegTest

popd > /dev/null
