#!/usr/bin/env bash

set -euo pipefail

echo "--- Executing Unit Tests"
sudo chmod -R go-w filebeat/

umask 0022
mage -d filebeat unitTest
