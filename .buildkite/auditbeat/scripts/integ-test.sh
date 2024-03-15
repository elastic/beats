#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Executing Integration Tests"
sudo chmod -R go-w auditbeat/

umask 0022
mage -d auditbeat build integTest
