#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/linux-env.sh
source .buildkite/filebeat/scripts/packaging/package-util.sh

IMG_POSTFIX="-SNAPSHOT"
VARIANTS=("" "-ubi" "-oss")
VERSION="$(make get-version)"
SOURCE_TAG+="${VERSION}${IMG_POSTFIX}"
BEAT_NAME="filebeat"
TARGET="observability-ci/${BEAT_NAME}"
# Remove following once beats fully migrated
BK_IMG_POSTFIX="-BK-SNAPSHOT"
BK_SOURCE_TAG+="${VERSION}${BK_IMG_POSTFIX}"

echo "--- Creating package"
mage -d filebeat package

echo "--- Distribution list"
ls -la filebeat/build/distributions

echo "--- Docker image list"
docker images

define_tags
check_is_arm

for variant in "${VARIANTS[@]}"; do

  source="beats/${BEAT_NAME}${variant}"

  for tag in "${tags[@]}"; do
  targetTag=$tag${is_arm}

  sourceName="${DOCKER_REGISTRY}/${source}:${SOURCE_TAG}"
  targetName="${DOCKER_REGISTRY}/${TARGET}:${targetTag}"

  if docker image inspect "${sourceName}" &>/dev/null; then
    echo "--- Tag & Push"
    echo "Source name: $sourceName"
    echo "Target name: $targetName"

    # Remove following lines once beats fully migrated
    bkSourceName="${DOCKER_REGISTRY}/${source}:${BK_SOURCE_TAG}"
    docker tag "$sourceName" "$bkSourceName"
    # Replace bkSourceName to sourceName once beats fully migrated
    docker tag "${bkSourceName}" "${targetName}"
#    docker push "${targetName}"
  else
    echo "Docker image ${sourceName} does not exist"
  fi
  done
done
