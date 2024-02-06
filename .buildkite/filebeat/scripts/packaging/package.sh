#!/usr/bin/env bash

set -euo pipefail

#source .buildkite/env-scripts/linux-env.sh
source .buildkite/env-scripts/util.sh

#echo "--- Creating package"
#mage -d filebeat package


TAGS=("${BUILDKITE_COMMIT}")

VARIANTS=("" "-ubi" "-oss")

# IMAGES
SOURCE_NAMESPACE="beats"
BEAT_NAME="filebeat"
SOURCE="beats/${BEAT_NAME}${variant}"
TARGET="observability-ci/${BEAT_NAME}"

# ARGS
REGISTRY="docker.elastic.co"
SNAPSHOT=true
VERSION="$(make get-version)",
images: images

#VARIANTS -> foreach = IMAGES: image map with SOURCE, TARGET, ARCH

#IMAGES -> foreach = tag and push
# registry: ${REGISTRY},
# sourceTag: calculate_tags->sourceTag,
# targetTag: "${tag}" (non arm) // ${tag}-${image.arch} (arm) --> foreach $TAGS
# source: ${SOURCE},
# target: ${TARGET}


define_tags() {
echo "--- Defined tags"
  if [ "$SNAPSHOT" = true ]; then
    aliasVersion="${VERSION%.*}${IMG_POSTFIX}"
    sourceTag+="${VERSION}${IMG_POSTFIX}"
  fi

  if is_pr; then
    TAGS+=("pr-${GITHUB_PR_NUMBER}")
  else
    TAGS+=("${sourceTag}" "${aliasVersion}")
  fi

  local tag=""
  for tag in "${TAGS[@]}"; do
    echo "$tag"
  done
}

#buildkite-agent annotate "Tag '$TAG' has been created." --style 'success' --context 'ctx-success'

echo "--- Calculating tags"
daefine_tags

#echo "--- Setting git config"
#set_git_config
