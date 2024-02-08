#!/usr/bin/env bash

set -euo pipefail

# Remove when custom image is set up
source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Integration Tests"
# Remove when custom image is set up
sudo chmod -R go-w heartbeat/

cd heartbeat
# Remove when custom image is set up
umask 0022
mage pythonIntegTest
