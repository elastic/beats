#!/usr/bin/env bash
set -euo pipefail

source .buildkite/scripts/ech.sh

BEAT_PATH=${1:?"Error: Specify the beat path: custom_fips_ech_test.sh [beat_path]"}

trap 'ech_down' EXIT

# TEMPORARY FIX: Use a fixed snapshot version until the snapshot versioning is fixed.
STACK_VERSION="9.2.0-SNAPSHOT"
# STACK_VERSION="$(./dev-tools/get_version)-SNAPSHOT"

ech_up $STACK_VERSION

echo "~~~ Running custom FIPS ECH tests"

pushd $BEAT_PATH
GOEXPERIMENT=systemcrypto SNAPSHOT=true FIPS=true mage build fipsECHTest
popd
