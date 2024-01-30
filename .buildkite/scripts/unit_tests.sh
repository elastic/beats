#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_project=$1

echo "--- Run Unit Tests"
sudo chmod -R go-w "${beats_project}/"
pushd "${beats_project}" > /dev/null
umask 0022
mage build unitTest
popd > /dev/null
