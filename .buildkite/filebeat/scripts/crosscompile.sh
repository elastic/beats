#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Crosscompile"
make -C filebeat crosscompile
