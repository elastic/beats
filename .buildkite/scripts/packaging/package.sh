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

echo "--- Docker image list"
docker images

define_tags

targetSuffix=""
if [[ ${HW_TYPE} == "aarch64" || ${HW_TYPE} == "arm64" ]]; then
  targetSuffix="-arm64"
fi

for variant in "${VARIANTS[@]}"; do
  source="beats/${BEATS_PROJECT_NAME}${variant}"

  for tag in "${tags[@]}"; do
    targetTag=$tag${targetSuffix}

    sourceName="${DOCKER_REGISTRY}/${source}:${SOURCE_TAG}"
    targetName="${DOCKER_REGISTRY}/${TARGET}:${targetTag}"
    #TODO Remove following line once beats fully migrated to Buildkite and Jenkins builds will be disabled
    #Avoid clashing with the Jenkins produced images
    targetName="${targetName}-buildkite"

    if docker image inspect "${sourceName}" &>/dev/null; then
      echo "--- Tag & Push with target: $targetName"
      echo "Source name: $sourceName"
      docker tag "$sourceName" "$targetName"
      docker push "$targetName"
    else
      echo "Docker image ${sourceName} does not exist"
    fi
  done
done
