#!/usr/bin/env bash

set -euo pipefail

echo "--- Running Unit Tests"
# Remove when custom image is set up
sudo chmod -R go-w heartbeat/

cd heartbeat
# Remove when custom image is set up
umask 0022
mage build unitTest
