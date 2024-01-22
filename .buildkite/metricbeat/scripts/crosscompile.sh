#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- Run Crosscompile"
make -C metricbeat crosscompile
