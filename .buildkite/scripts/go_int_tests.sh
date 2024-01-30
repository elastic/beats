#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_project=$1

echo "--- Run Go Intergration Tests for $beats_project"
pushd "${beats_project}" > /dev/null

mage goIntegTest

popd > /dev/null
