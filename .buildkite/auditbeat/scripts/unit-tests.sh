#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/unix-env.sh

echo ":: Running Unit Tests ::"
sudo chmod -R go-w auditbeat/

cd auditbeat
umask 0022
mage build unitTest
