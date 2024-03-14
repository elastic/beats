#!/usr/bin/env bash

set -euo pipefail

#source .buildkite/scripts/packaging/package-util.sh
#source .buildkite/env-scripts/util.sh

IMG_POSTFIX="-SNAPSHOT"
VARIANTS=("" "-ubi" "-oss")
VERSION="$(make get-version)"
SOURCE_TAG+="${VERSION}${IMG_POSTFIX}"
TARGET="observability-ci/${BEATS_PROJECT_NAME}"

#echo "--- Git version: $(git --version)"
#echo "--- Mage version: $(mage -version)"
#echo "--- Go version: $(go version)"
#echo "--- Make version: $(make get-version)"
#echo "--- Go modules: $(go list -m all)"
#echo "--- GCC version: $(gcc --version)"
#echo "--- G++ version: $(g++ --version)"
#
#echo "--- Adding bin path"
#add_bin_path

#echo "--- With GO"
#with_go

#echo "--- With Mage: $(with_mage)"

echo "--- Creating package"
mage -d "${BEATS_PROJECT_NAME}" package
#
#echo "--- Distribution list"
#dir="${BEATS_PROJECT_NAME}/build/distributions"
#buildkite-agent artifact upload "$dir/*.tar.gz;$dir/*.tar.gz.sha512"
#
#echo "--- Docker image list"
#docker images
#
#define_tags
#
#targetSuffix=""
#if [[ ${HW_TYPE} == "aarch64" || ${HW_TYPE} == "arm64" ]]; then
#  targetSuffix="-arm64"
#fi
#
#for variant in "${VARIANTS[@]}"; do
#  source="beats/${BEATS_PROJECT_NAME}${variant}"
#
#  for tag in "${tags[@]}"; do
#    targetTag=$tag${targetSuffix}
#
#    sourceName="${DOCKER_REGISTRY}/${source}:${SOURCE_TAG}"
#    targetName="${DOCKER_REGISTRY}/${TARGET}:${targetTag}"
#    # Remove following line once beats fully migrated
#    targetName="${targetName}-buildkite"
#
#    if docker image inspect "${sourceName}" &>/dev/null; then
#      echo "--- Tag & Push with target: $targetName"
#      echo "Source name: $sourceName"
#      docker tag "$sourceName" "$targetName"
#      docker push "$targetName"
#    else
#      echo "Docker image ${sourceName} does not exist"
#    fi
#  done
#done
