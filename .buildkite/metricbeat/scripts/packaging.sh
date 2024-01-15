#!/usr/bin/env bash

set -euo pipefail

source .buildkite/metricbeat/scripts/common.sh

echo ":: Evaluate Filebeat Changes ::"

changeset="^metricbeat/.*
    ^@oss"

if ! are_files_changed "$changeset" ; then
  message="No files changed within metricbeat changeset"
  echo "$message"
  buildkite-agent annotate "$message" --style "info" --context "$BUILDKITE_STEP_KEY"
  exit 0
fi

echo "--- prepare env"
add_bin_path
with_go ${GO_VERSION}
with_mage
with_python

echo "--- Packaging"
pushd "metricbeat" > /dev/null
# chmod -R go-w ./mb/testdata/
# umask 0022
mage package
popd > /dev/null
