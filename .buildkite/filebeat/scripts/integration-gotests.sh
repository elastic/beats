#!/usr/bin/env bash

set -euo pipefail

echo "--- Executing Integration Tests"
sudo chmod -R go-w filebeat/

cd filebeat
umask 0022
mage goIntegTest
