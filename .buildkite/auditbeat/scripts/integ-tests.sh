#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Running Integration Tests"
echo "using kernel $(uname -r)"
sudo chmod -R go-w auditbeat/

cd auditbeat
mage build integTest
