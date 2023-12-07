#!/usr/bin/env bash

set -euo pipefail

source .buildkite/filebeat/scripts/common.sh

# ToDo - remove after Beats agent is created"
echo ":: Setup Env ::"
add_bin_path
with_go
with_mage
# ToDo - end

echo ":: Execute Integration Tests ::"
sudo chmod -R go-w filebeat/

cd filebeat
umask 0022
mage goIntegTest
