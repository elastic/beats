#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- prepare env"
add_bin_path
with_go ${GO_VERSION}
with_mage


echo "--- run unit tests"
pushd "metricbeat" > /dev/null
chmod -R go-w ./mb/testdata/
#umask 0022
mage build unitTest
popd > /dev/null
