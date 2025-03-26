#!/usr/bin/env bash

# This script is used by GH workflows to execute tests selectively, based on changes.
# It dynamically generates the matrix based on following logic:
# 1. if changes are in a directory(ies) listed in a CHANGESET_FILE, it/those will be added to a matrix
# 2. if changed directories are not in the list - tests will be skipped
# 3. if the workflow related files were changed - tests for all directories listed in CHANGESET_FILE will be triggered

# Path to file with a list of directories to track changes
CHANGESET_FILE=${1:?"CHANGESET_FILE parameter not set"}
# Array of OS versions (GH runner names) to execute tests
OS_VERSIONS=${2:?"OS_VERSIONS parameter not set"}
# Name of a workflow file to track if changed
WORKFLOW_FILE=${3:?"WORKFLOW_FILE parameter not set"}

readarray -t changeset <$CHANGESET_FILE
changeList=$(git diff --name-only HEAD~1)
changedDirs=($(echo "$changeList" | sed 's/^[ 0-9.]\+% //g' | awk -F/ '{print $1}' | sort -u))

# Filter changed directories based on a changeset
beats=()
for dir in "${changedDirs[@]}"; do
  if [[ "${changeset[*]}" =~ $dir ]]; then
    beats+=("$dir")
  fi
done

# If the workflow changed - run all tests, no matter of what was the other changes
if [[ $changeList == *"$WORKFLOW_FILE"* || $changeList == *"macos.d/generate-matrix.sh"* ]]; then
  matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${changeset[@]}" | jq -R . | jq -s .)" \
    --argjson osArray "$(printf '%s\n' "${OS_VERSIONS[@]}" | jq -R . | jq -s .)" \
    '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

  echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT

# Check if the changes affected beats
# if yes - generate a matrix to run tests only for changed beats
# if not - skip the following jobs
elif [[ "${#beats[@]}" != 0 ]]; then
    matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${beats[@]}" | jq -R . | jq -s .)" \
      --argjson osArray "$(printf '%s\n' "${OS_VERSIONS[@]}" | jq -R . | jq -s .)" \
      '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

    echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT

# Skip tests if changes are not related to directories listed in the changeset or not related to workflow files
else
    echo "No changes detected, tests will be skipped"
fi
