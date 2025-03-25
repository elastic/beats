#!/usr/bin/env bash

readarray -t whitelist < ".github/workflows/macos.d/values.d/whitelisted-dirs"
changeList=$(git diff --name-only HEAD~1)
changedDirs=($(echo "$changeList" | sed 's/^[ 0-9.]\+% //g' | awk -F/ '{print $1}' | sort -u))

beats=()
for dir in "${changedDirs[@]}"; do
  if [[ "${whitelist[*]}" =~ $dir ]]; then
    beats+=("$dir")
  fi
done

# If the workflow changed - run all tests, no matter of what was the other changes
if [[ $changeList == *"macos-unit-tests-pr.yml"* ]]; then
  osArray=("${MACOS_13_X64}" "${MACOS_14_ARM}")
  matrix=$(jq -n --argjson dirs "$(printf '%s\n' "${beats[@]}" | jq -R . | jq -s .)" \
    --argjson osArray "$(printf '%s\n' "${osArray[@]}" | jq -R . | jq -s .)" \
    '[ $dirs[] as $dir | $osArray[] as $os | {beat: $dir, os: $os} ]')
fi

# Check if the changes were done in beats
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
