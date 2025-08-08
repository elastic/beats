#!/usr/bin/env bash
set -euo pipefail

source .buildkite/scripts/ech.sh

BEAT_PATH=${1:?"Error: Specify the beat path: custom_fips_ech_test.sh [beat_path]"}

trap 'ech_down' EXIT

# We manually override the stack version during the period when we need to bump
# the Beats version in `libbeat/version/version.go` but new artifacts with that bumped version
# haven't been published yet and are, therefore, not yet available in ECH.
# Once artifacts matching the version in `libbeat/version/version.go` are available in ECH, the
# following line should be removed and the line after that should be uncommented.
STACK_VERSION="9.1.1-SNAPSHOT"
# STACK_VERSION="$(./dev-tools/get_version)-SNAPSHOT"

ech_up $STACK_VERSION

echo "~~~ Running custom FIPS ECH tests"

pushd $BEAT_PATH
GOEXPERIMENT=systemcrypto SNAPSHOT=true FIPS=true mage build fipsECHTest
popd
