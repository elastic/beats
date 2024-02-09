#!/usr/bin/env bash

set -euo pipefail

#source .buildkite/env-scripts/linux-env.sh
source .buildkite/env-scripts/util.sh

IMG_POSTFIX="-BK-SNAPSHOT"
VARIANTS=("" "-ubi" "-oss")
VERSION="$(make get-version)"
SOURCE_TAG+="${VERSION}${IMG_POSTFIX}"
BEAT_NAME="filebeat"
TARGET="observability-ci/${BEAT_NAME}"

define_tags() {
  aliasVersion="${VERSION%.*}${IMG_POSTFIX}"
  tags=("${BUILDKITE_COMMIT}")

  if is_pr; then
    tags+=("pr-${GITHUB_PR_NUMBER}")
  else
    tags+=("${SOURCE_TAG}" "${aliasVersion}")
  fi
}

check_is_arm() {
  if [[ ${HW_TYPE} == "aarch64" || ${HW_TYPE} == "arm64" ]]; then
    is_arm="-arm"
  else
    is_arm=""
  fi
}

define_tags

for variant in "${VARIANTS[@]}"; do
  echo "--- PARAMS for variant: $variant"

  check_is_arm
  registry=${DOCKER_REGISTRY}
  sourceTag=$SOURCE_TAG
  source="beats/${BEAT_NAME}${variant}"
  target=$TARGET

  echo "Registry: $registry"
  echo "Source: $source"
  echo "Source tag: $sourceTag"
  echo "Target: $target"

  for tag in "${tags[@]}"; do
    targetTag=$tag${is_arm}
    echo "Target tag: $targetTag"
  done
done

#echo "--- Creating package"
#mage -d filebeat package

#echo "--- Setting git config"
#set_git_config

#buildkite-agent annotate "Tag '$TAG' has been created." --style 'success' --context 'ctx-success'
