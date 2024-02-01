#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh

echo ":: Start Packaging ::"
#cd filebeat
#umask 0022
mage -d filebeat package
