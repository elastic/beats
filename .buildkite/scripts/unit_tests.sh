#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfolder=$1

sudo chmod -R go-w ${beats_subfolder}/

echo "--- Run Unit Tests"
pushd "${beats_subfolder}" > /dev/null

umask 0022
mage build unitTest

popd > /dev/null
