#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

beats_project=$1

echo "--- Run Crosscompile for $beats_project"
make -C "${beats_project}" crosscompile
