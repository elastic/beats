#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- Run Unit Tests"
sudo chmod -R go-w metricbeat/
pushd "metricbeat" > /dev/null
umask 0022
mage build unitTest
popd > /dev/null
