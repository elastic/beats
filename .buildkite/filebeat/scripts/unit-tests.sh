#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh
source .buildkite/env-scripts/macos-env.sh

if [[ $PLATFORM_TYPE == Darwin* ]]; then
  echo ":: ULIMIT :: $(ulimit -n)"
fi

echo ":: Execute Unit Tests ::"
sudo chmod -R go-w filebeat/

cd filebeat
umask 0022
mage build unitTest
