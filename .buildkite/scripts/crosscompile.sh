#!/usr/bin/env bash

set -euo pipefail

echo "--- Run Crosscompile for $BEATS_PROJECT_NAME"
make -C "${BEATS_PROJECT_NAME}" crosscompile
