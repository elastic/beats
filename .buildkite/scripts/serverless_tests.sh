#!/usr/bin/env bash
#
# Packages a Beat and runs its serverless integration tests against a
# short-lived Elastic Cloud serverless project, which the tests create and delete.
# Requires EC_API_KEY.
#
# Usage: serverless_tests.sh <beat_path>   (for example x-pack/filebeat)
set -euo pipefail

BEAT_PATH=${1:?"Error: Specify the beat path: serverless_tests.sh [beat_path]"}
BEAT_NAME=$(basename "${BEAT_PATH}")
: "${EC_API_KEY:?"Error: EC_API_KEY must be set"}"

pushd "${BEAT_PATH}"

echo "~~~ Packaging ${BEAT_NAME}"
# The package carries the kibana dashboards and modules the setup commands need.
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

echo "~~~ Running serverless tests for ${BEAT_NAME}"
mage serverlessTest

popd
