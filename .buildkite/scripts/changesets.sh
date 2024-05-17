#!/usr/bin/env bash

set -euo pipefail

OSS_MODULE_PATTERN="^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
XPACK_MODULE_PATTERN="^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"

definePattern() {
  pattern="${OSS_MODULE_PATTERN}"

  if [[ "$beatPath" == *"x-pack/"* ]]; then
    pattern="${XPACK_MODULE_PATTERN}"
  fi
}

defineExclusions() {
  exclude="^$beatPath\/module\/(.*(?<!\.asciidoc|\.png))$"
}

defineFromCommit() {
  local previousCommit
  local changeTarget=${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-$BUILDKITE_BRANCH}

  previousCommit=$(git rev-parse HEAD^)

  from=${changeTarget:+"origin/$changeTarget"}
  from=${from:-$previousCommit}
  from=${from:-$BUILDKITE_COMMIT}
}

getMatchingModules() {
  local changedPaths
  mapfile -t changedPaths < <(git diff --name-only "$from"..."$BUILDKITE_COMMIT" | grep -P "$exclude" | grep -oP "$pattern")
  mapfile -t modulesMatched < <(printf "%s\n" "${changedPaths[@]}" | grep -o 'module/.*' | awk -F '/' '{print $2}' | sort -u)
}

addToModuleEnvVar() {
  local module
  for module in "${modulesMatched[@]}"; do
    if [[ -z ${modules+x} ]]; then
      modules="$module"
    else
      modules+=",$module"
    fi
  done
}

defineModuleFromTheChangeSet() {
  beatPath=$1

  definePattern
  defineExclusions
  defineFromCommit
  getMatchingModules

  if [ "${#modulesMatched[@]}" -gt 0 ]; then
    addToModuleEnvVar
    export MODULE=$modules
  else
    echo "~~~ No changes in modules for $beatPath"
  fi
}
