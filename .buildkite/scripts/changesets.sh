#!/usr/bin/env bash

# This script contains helper functions related to what should be run depending on Git changes

are_paths_changed() {
  local patterns=("${@}")
  local changelist=()
  for pattern in "${patterns[@]}"; do
    changed_files=($(git diff --name-only HEAD@{1} HEAD | grep -E "$pattern"))
    if [ "${#changed_files[@]}" -gt 0 ]; then
      changelist+=("${changed_files[@]}")
    fi
  done

  if [ "${#changelist[@]}" -gt 0 ]; then
    echo "Files changed:"
    echo "${changelist[*]}"
    return 0
  else
    echo "No files changed within specified changeset:"
    echo "${patterns[*]}"
    return 1
  fi
}

are_changed_only_paths() {
  local patterns=("${@}")
  local changed_files=($(git diff --name-only HEAD@{1} HEAD))
  local matched_files=()
  for pattern in "${patterns[@]}"; do
    local matched=($(grep -E "${pattern}" <<< "${changed_files[@]}"))
    if [ "${#matched[@]}" -gt 0 ]; then
      matched_files+=("${matched[@]}")
    fi
  done
  if [ "${#matched_files[@]}" -eq "${#changed_files[@]}" ] || [ "${#changed_files[@]}" -eq 0 ]; then
    return 0
  fi
  return 1
}

defineModuleFromTheChangeSet() {
  # This function sets a `MODULE` env var, required by IT tests, containing a comma separated list of modules for a given beats project (specified via the first argument).
  # The list is built depending on directories that have changed under `modules/` excluding anything else such as asciidoc and png files.
  # `MODULE` will empty if no changes apply.
  local project_path=$1
  local project_path_transformed=$(echo "$project_path" | sed 's/\//\\\//g')
  local project_path_exclussion="((?!^${project_path_transformed}\\/).)*\$"
  local exclude=("^(${project_path_exclussion}|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)")

  if [[ "$project_path" == *"x-pack/"* ]]; then
    local pattern=("$XPACK_MODULE_PATTERN")
  else
    local pattern=("$OSS_MODULE_PATTERN")
  fi
  local changed_modules=""
  local module_dirs=$(find "$project_path/module" -mindepth 1 -maxdepth 1 -type d)
  for module_dir in $module_dirs; do
    if are_paths_changed $module_dir && ! are_changed_only_paths "${exclude[@]}"; then
      if [[ -z "$changed_modules" ]]; then
        changed_modules=$(basename "$module_dir")
      else
        changed_modules+=",$(basename "$module_dir")"
      fi
    fi
  done

  if [[ -z "$changed_modules" ]]; then # TODO: remove this conditional when issue https://github.com/elastic/ingest-dev/issues/2993 gets resolved
    if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
      export MODULE="aws"
    else
      export MODULE="kubernetes"
    fi
  else
    export MODULE="${changed_modules}"  # TODO: once https://github.com/elastic/ingest-dev/issues/2993 gets resolved, this should be the only thing we export
  fi
}
