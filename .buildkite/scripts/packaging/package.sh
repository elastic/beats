#!/usr/bin/env bash

set -euo pipefail

source .buildkite/scripts/packaging/package-util.sh

IMG_POSTFIX="-SNAPSHOT"
VARIANTS=("" "-ubi" "-oss")
VERSION="$(make get-version)"
SOURCE_TAG+="${VERSION}${IMG_POSTFIX}"
TARGET="observability-ci/${BEATS_PROJECT_NAME}"

echo "--- Creating package"
mage -d "${BEATS_PROJECT_NAME}" package

echo "--- Distribution list"
dir="${BEATS_PROJECT_NAME}/build/distributions"
buildkite-agent artifact upload "$dir/*.tar.gz;$dir/*.tar.gz.sha512"

