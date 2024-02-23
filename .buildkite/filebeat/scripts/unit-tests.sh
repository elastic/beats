#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Unit Tests"
sudo chmod -R go-w filebeat/

umask 0022
mage -d filebeat unitTest
