#!/bin/bash

set -euo pipefail

echo "--- Pre install"
source .buildkite/scripts/pre-install-command.sh
add_bin_path
with_mage

echo "--- Mage notice"
mage notice

echo "--- Mage check"
mage -v check
