#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- run unit tests"
pushd "metricbeat" > /dev/null
chmod -R go-w ./mb/testdata/
#umask 0022
mage build unitTest
popd > /dev/null
