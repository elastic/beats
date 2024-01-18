#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/unix-env.sh

echo ":: Evaluate Auditbeat Changes ::"

echo ":: Start Packaging ::"
cd auditbeat
umask 0022
mage package

