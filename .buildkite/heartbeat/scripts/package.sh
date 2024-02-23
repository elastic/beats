#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Docker Version: $(docker --version)"

echo "--- Start Packaging"
cd heartbeat
umask 0022
mage package

