#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfolder=$1

sudo chmod -R go-w ${beats_subfolder}/

echo "--- Run Go Intergration Tests for $beats_subfolder"
pushd "${beats_subfolder}" > /dev/null

umask 0022
mage goIntegTest

popd > /dev/null
