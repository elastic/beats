#!/usr/bin/env bash

# This script is used by GH actions to dynamically generate the matrix for the workflow,
# allowing to run tests selectively based on changes.

# Array of OS versions (GH runner names) to execute tests
OS_VERSIONS=${1:?"OS_VERSIONS parameter not set"}
# Name of a workflow file to track if changed
WORKFLOW=${WORKFLOW:?"WORKFLOW parameter not set"}

os=("${OS_VERSIONS}")
changeList=$(git diff --name-only HEAD~1)
changedDirs=($(echo "$changeList" | sed 's/^[ 0-9.]\+% //g' | awk -F/ '{print $1}' | sort -u))

# If the workflow changed - run all tests, no matter of what were the other changes
if [[ $changeList == *"$WORKFLOW"* || $changeList == *"test-matrix-composite-action"* ]]; then
  # Since GH actions do not support accessing the `paths` section in a runtime, get required beats manually
  beats=($(awk '/paths:/ {flag=1; next} /^[^ ]/ {flag=0} flag && $2 !~ /^\.github/ {print $2}'))
  matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${beats[@]}" | jq -R . | jq -s .)" \
    --argjson osArray "$(printf '%s\n' "${os[@]}" | jq -R . | jq -s .)" \
    '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

  echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT
else
    matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${changedDirs[@]}" | jq -R . | jq -s .)" \
      --argjson osArray "$(printf '%s\n' "${os[@]}" | jq -R . | jq -s .)" \
      '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

    echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT
fi
