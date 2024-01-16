#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- Prepare env"
ulimit -Sn 50000
add_bin_path
with_go ${GO_VERSION}
with_mage
with_python

echo "--- Run unit tests"
sudo chmod -R go-w metricbeat/
pushd "metricbeat" > /dev/null
umask 0022
mage build unitTest
popd > /dev/null
