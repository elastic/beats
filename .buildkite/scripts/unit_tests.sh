#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Unit Tests"
sudo chmod -R go-w "${BEATS_PROJECT_NAME}/"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

umask 0022
mage build unitTest

popd > /dev/null
