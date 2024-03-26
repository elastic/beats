#!/usr/bin/env bash

set -euo pipefail

is_pr() {
  if [[ $BUILDKITE_PULL_REQUEST != false ]]; then
    return 0
  else
    return 1
  fi
}

define_tags() {
  aliasVersion="${VERSION%.*}${IMG_POSTFIX}"
  tags=("${BUILDKITE_COMMIT}")

  if is_pr; then
    tags+=("pr-${GITHUB_PR_NUMBER}")
  else
    tags+=("${SOURCE_TAG}" "${aliasVersion}")
  fi
}
