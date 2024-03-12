#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo "--- Git status"
git --no-pager status || true
git --no-pager log -n2 || true
git --no-pager diff || true

echo "--- Start Packaging"
cd filebeat
umask 0022
mage package
