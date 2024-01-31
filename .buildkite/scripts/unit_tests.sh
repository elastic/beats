#!/bin/bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Unit Tests"

#sudo command doesn't work at the "pre-command" hook because of another user environment (root with strange permissions)
sudo chmod -R go-w "${BEATS_PROJECT_NAME}/"     #fix the fulesystem permissions issue like this:https://buildkite.com/elastic/beats-metricbeat/builds/1154#018d12db-dc0c-4bcd-b9b4-d5dece0b42c6/272-1267

pushd "${BEATS_PROJECT_NAME}" > /dev/null

umask 0022    # fix the filesystem permissions issue like this: https://buildkite.com/elastic/beats-metricbeat/builds/1329#018d3179-25a9-475b-a2c8-64329dfe092b/320-1696
mage build unitTest

popd > /dev/null
