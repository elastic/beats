#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Crosscompile for $beats_subfilder"
pushd "${beats_subfilder}" > /dev/null

make -C $beats_subfilder crosscompile

popd > /dev/null
