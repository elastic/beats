#!/usr/bin/env bash

set -euo pipefail

echo "--- Run Packaging for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage package

popd > /dev/null
