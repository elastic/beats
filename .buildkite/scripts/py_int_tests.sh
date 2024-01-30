#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_project=$1

echo "--- Run Python Intergration Tests for $beats_project"
pushd "${beats_project}" > /dev/null

mage pythonIntegTest

popd > /dev/null
