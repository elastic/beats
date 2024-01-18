#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/unix-env.sh

echo ":: Execute Integration Tests ::"
sudo chmod -R go-w filebeat/

cd filebeat
umask 0022
mage goIntegTest
