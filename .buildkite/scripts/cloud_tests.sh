#!/usr/bin/env bash

# What Terraform Module will run
export MODULE_DIR="x-pack/metricbeat/module/aws"

source .buildkite/scripts/install_tools.sh

set -euo pipefail

trap 'teardown; unset_secrets' EXIT

# Set the MODULE env variable if possible
defineModuleFromTheChangeSet "${MODULE_DIR}"

# Prepare the cloud resources using Terraform
startCloudTestEnv "${MODULE_DIR}"

# Run tests
echo "--- Run Cloud Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

mage build test

popd > /dev/null
