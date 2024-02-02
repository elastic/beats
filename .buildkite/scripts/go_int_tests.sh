#!/bin/bash

set -euo pipefail

echo "--- Run Go Intergration Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage goIntegTest

popd > /dev/null
