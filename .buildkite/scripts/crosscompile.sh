#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfolder=$1

echo "--- Run Crosscompile for $beats_subfolder"

pushd "${beats_subfolder}" > /dev/null

make crosscompile

popd > /dev/null
