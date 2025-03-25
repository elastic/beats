#!/usr/bin/env bash

readarray -t whitelist <".github/workflows/macos.d/values.d/beats-macos"
changeList=$(git diff --name-only HEAD~1)
changedDirs=($(echo "$changeList" | sed 's/^[ 0-9.]\+% //g' | awk -F/ '{print $1}' | sort -u))

beats=()
for dir in "${changedDirs[@]}"; do
  if [[ "${whitelist[*]}" =~ $dir ]]; then
    beats+=("$dir")
  fi
done

osArray=("${MACOS_13_X64}" "${MACOS_14_ARM}")
# If the workflow changed - run all tests, no matter of what was the other changes
if [[ $changeList == *"macos-unit-tests-pr.yml"* || $changeList == *"macos.d/generate-matrix.sh"* ]]; then
  matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${whitelist[@]}" | jq -R . | jq -s .)" \
    --argjson osArray "$(printf '%s\n' "${osArray[@]}" | jq -R . | jq -s .)" \
    '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

  echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT
else
  # Check if the changes affected beats
  # if yes - generate a matrix to run tests only for changed beats
  # if not - skip the following jobs
  if [[ "${#beats[@]}" != 0 ]]; then
    matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${beats[@]}" | jq -R . | jq -s .)" \
      --argjson osArray "$(printf '%s\n' "${osArray[@]}" | jq -R . | jq -s .)" \
      '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')

    echo "matrix={\"include\":$(echo $matrix)}" >>$GITHUB_OUTPUT
  else
    echo "No changes detected, tests will be skipped"
  fi
fi
