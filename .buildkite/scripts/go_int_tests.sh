#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Go Intergration Tests for $beats_subfilder"
pushd "${beats_subfilder}" > /dev/null

mage goIntegTest

popd > /dev/null
