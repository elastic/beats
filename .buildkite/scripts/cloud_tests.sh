#!/usr/bin/env bash

# What Terraform Module will run
if [[ "$BUILDKITE_STEP_KEY" == "xpack-metricbeat-pipeline" ]]; then
  export MODULE_DIR="x-pack/metricbeat/module/aws"
elif [["$BUILDKITE_STEP_KEY" == "xpack-filebeat-pipeline"]]; then
  export MODULE_DIR="x-pack/filebeat/input/awss3/_meta/terraform"
fi

source .buildkite/scripts/install_tools.sh

set -euo pipefail

trap 'teardown || true; unset_secrets' EXIT

# Prepare the cloud resources using Terraform
startCloudTestEnv "${MODULE_DIR}"

# Run tests
echo "--- Run Cloud Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage build test

popd > /dev/null
