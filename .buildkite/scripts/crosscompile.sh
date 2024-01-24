#!/bin/bash

source .buildkite/scripts/common.sh

set -euo pipefail

beats_subfilder=$1

echo "--- Run Crosscompile for $beats_subfilder"
make -C "${beats_subfilder}" crosscompile
