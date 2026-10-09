#!/usr/bin/env bash
set -euo pipefail

source .buildkite/scripts/serverless.sh

BEAT_PATH=${1:?"Error: Specify the beat path: serverless_tests.sh [beat_path]"}
BEAT_NAME=$(basename "${BEAT_PATH}")

trap 'serverless_down' EXIT

# Package before creating the project, so it isn't kept around while packaging runs.
# The package carries the kibana dashboards and modules the setup commands need.
echo "~~~ Packaging ${BEAT_NAME}"
pushd "${BEAT_PATH}"
SNAPSHOT=true PLATFORMS=linux/amd64 PACKAGES=tar.gz mage package

# Match exactly one tarball: a stale or extra package must not be picked up silently.
shopt -s nullglob
tarballs=(build/distributions/"${BEAT_NAME}"-*-linux-x86_64.tar.gz)
shopt -u nullglob
if [[ ${#tarballs[@]} -ne 1 ]]; then
  echo "Expected exactly one ${BEAT_NAME} linux tarball in build/distributions, found: ${tarballs[*]:-none}" >&2
  exit 1
fi

BEAT_HOME="$(pwd)/build/serverless-home"
rm -rf "${BEAT_HOME}"
mkdir -p "${BEAT_HOME}"
tar -xzf "${tarballs[0]}" -C "${BEAT_HOME}" --strip-components=1
export BEAT_HOME
popd

serverless_up

echo "~~~ Running serverless tests"

pushd "${BEAT_PATH}"
mage serverlessTest
popd
