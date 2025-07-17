#!/usr/bin/env bash
set -euo pipefail

source .buildkite/scripts/ech.sh

BEAT_PATH=$1

if [ -z "$BEAT_PATH" ]; then
    echo "Error: Specify the beat path: custom_fips_ech_test.sh [beat_path]" >&2
    exit 1
fi

trap 'ech_down' EXIT

STACK_VERSION="$(./dev-tools/get_version)-SNAPSHOT"

ech_up $STACK_VERSION

echo "~~~ Running custom FIPS ECH tests"

pushd $BEAT_PATH
SNAPSHOT=true FIPS=true mage fipsECHTest
popd
