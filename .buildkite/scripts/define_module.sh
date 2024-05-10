#!/usr/bin/env bash

set -euo pipefail

OSS_MODULE_PATTERN="^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
XPACK_MODULE_PATTERN="^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
BEAT_PATH=$1
MODULE=''

definePattern() {
  pattern="${XPACK_MODULE_PATTERN}"

  if [[ "$BEAT_PATH" == *"x-pack/"* ]]; then
    pattern="${OSS_MODULE_PATTERN}"
  fi
}

defineExclusions() {
  local transformedDirectory=${BEAT_PATH//\//\\\/}
  local exclusion="((?!^${transformedDirectory}\\/).)*\$"
  exclude="^(${exclusion}|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)"
}

getGitMatchingGroup() {
  local previousCommit
  local matches
  local match

  local changeTarget=${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-$BUILDKITE_BRANCH}
  local baseCommit=$BUILDKITE_COMMIT
  previousCommit=$(git rev-parse HEAD^)
  from=$(echo ${changeTarget:+"origin/$changeTarget"}${previousCommit:-$baseCommit})
  to=$baseCommit

  matches=$(git diff --name-only "$from"..."$to" | grep -v "$exclude" | grep -oP "$pattern" | sort -u)

  match=$(echo "$matches" | wc -w)

  if [ "$match" -eq 1 ]; then
    echo "$matches"
  else
    echo ''
  fi
}

defineModule() {
  cd "${BEAT_PATH}"
  module=$(getGitMatchingGroup "$pattern" "$exclude")
  if [ ! -f "$BEAT_PATH/module/${module}" ]; then
    module=''
  fi
  cd - >/dev/null

  echo "${module}"
}
