#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Go Intergration Tests for $beats_subfilder"
sudo chmod -R go-w ${beats_subfilder}/
pushd "${beats_subfilder}" > /dev/null
# umask 0022
mage goIntegTest

popd > /dev/null
