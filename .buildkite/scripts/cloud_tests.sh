#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

trap 'cleanup; unset_secrets' EXIT

echo "--- Run Cloud Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

# Cloud stage for x-pack/metricbeat
if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
  if [[ "${BUILDKITE_STEP_KEY}" == "mandatory-int-test" || "${BUILDKITE_STEP_KEY}" == "mandatory-python-int-test" || "${BUILDKITE_STEP_KEY}" == "extended-cloud-test" ]]; then
    # Configure MODULE env variable.
    defineModuleFromTheChangeSet "${MODULE_DIR}"
  fi
  if [[ "${BUILDKITE_STEP_KEY}" == "extended-cloud-test" ]]; then
    # Start the required services for the required
    startCloudTestEnv "${MODULE_DIR}"
  fi
fi

mage build test

popd > /dev/null
