#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Unit Tests"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage build unitTest

popd > /dev/null
