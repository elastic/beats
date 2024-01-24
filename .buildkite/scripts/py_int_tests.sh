#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Python Intergration Tests for $beats_subfilder"
pushd "${beats_subfilder}" > /dev/null

mage pythonIntegTest

popd > /dev/null
