#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Integration Tests"
sudo chmod -R go-w filebeat/

cd filebeat
umask 0022
mage pythonIntegTest
