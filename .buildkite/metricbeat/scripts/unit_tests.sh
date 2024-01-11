#!/bin/bash

source .buildkite/metricbeat/scripts/common.sh

set -euo pipefail

echo "--- prepare env"
ulimit -Sn 50000
add_bin_path
with_go ${GO_VERSION}
with_mage
with_python

if [ "$(uname)" == "Darwin" ]; then
  diskutil info -all
else
  echo "--- run unit tests"
  pushd "metricbeat" > /dev/null
  chmod -R go-w ./mb/testdata/
  #umask 0022
  mage build unitTest
  popd > /dev/null
fi