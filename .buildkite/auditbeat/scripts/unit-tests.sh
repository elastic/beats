#!/usr/bin/env bash

set -euo pipefail

echo "--- Running Unit Tests"
sudo chmod -R go-w auditbeat/

umask 0022
mage -d auditbeat build unitTest
