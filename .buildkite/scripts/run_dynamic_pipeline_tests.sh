#!/usr/bin/env bash
# Run tests for the dynamic pipeline generator only if it's a PR and related files have been changed
# this will allow us to fail fast, if e.g. a PR has broken the generator

set -euo pipefail

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

pipeline_generator_changeset=(
  "^.buildkite/pipeline.py"
)

if ! are_paths_changed "${pipeline_generator_changeset[@]}" || [[ "${BUILDKITE_PULL_REQUEST}" == "false" ]]; then
  echo "~~~ Skipping pipeline generator tests"
  exit
fi

echo "~~~ Execute pipeline generator tests"

python3 -mpip install --quiet "pytest"
pushd .buildkite
${ASDF_DIR}/installs/python/${ASDF_PYTHON_VERSION}/bin/pytest .
pytest .
popd
