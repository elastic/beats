#!/usr/bin/env bash

set -euo pipefail

echo "--- Run Unit Tests"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage build unitTest

popd > /dev/null
