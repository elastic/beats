#!/usr/bin/env bash
# This script contains helper functions related to what should be run depending on Git changes

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
  exclude="^$beatPath\/module\/(.*(?<!\.asciidoc|\.png|Dockerfile))$"
}

defineFromCommit() {
  local changeTarget=${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-$BUILDKITE_BRANCH}

  if [[ -z ${changeTarget+x} ]]; then
    # If not a PR (no target branch) - use last commit
    from=$(git rev-parse HEAD^)
  else
    # If it's a PR - add "origin/"
    from="origin/$changeTarget"
  fi
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
    echo "~~~ Detected file changes for some modules. Setting env var MODULE to [$MODULE]"
    echo "~~~ Resuming commands"
  else
    echo "~~~ No changes in modules for $beatPath"
  fi
}
