#!/usr/bin/env bash

set -euo pipefail

source .buildkite/metricbeat/scripts/common.sh

echo "--- Evaluate Metricbeat Changes"

changeset="^metricbeat/.*
    ^@oss"

if ! are_files_changed "$changeset" ; then
  message="No files changed within Metricbeat changeset"
  echo "$message"
  buildkite-agent annotate "$message" --style "info" --context "$BUILDKITE_STEP_KEY"
  exit 0
fi

echo "--- Packaging"
pushd "metricbeat" > /dev/null
mage package
popd > /dev/null
