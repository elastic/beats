#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Integration Tests"
sudo chmod -R go-w heartbeat/

cd heartbeat
umask 0022
mage goIntegTest
