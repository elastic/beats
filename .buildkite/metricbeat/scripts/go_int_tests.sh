#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- Run Intergration Tests"
pushd "metricbeat" > /dev/null

mage goIntegTest

popd > /dev/null
