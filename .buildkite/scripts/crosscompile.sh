#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Crosscompile for $BEATS_PROJECT_NAME"
make -C "${BEATS_PROJECT_NAME}" crosscompile
