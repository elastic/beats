#!/usr/bin/env bash

set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
NMAP_WIN_VERSION="7.12"           # Earlier versions of NMap provide WinPcap (the winpcap packages don't install nicely because they pop-up a UI)
GO_VERSION=$(cat .go-version)
ASDF_MAGE_VERSION="1.15.0"
PACKAGING_PLATFORMS="+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
PACKAGING_ARM_PLATFORMS="linux/arm64"

export SETUP_GVM_VERSION
export DOCKER_COMPOSE_VERSION
export DOCKER_COMPOSE_VERSION_AARCH64
export SETUP_WIN_PYTHON_VERSION
export NMAP_WIN_VERSION
export GO_VERSION
export ASDF_MAGE_VERSION
export PACKAGING_PLATFORMS
export PACKAGING_ARM_PLATFORMS

exportVars() {
  local platform_type="$(uname)"
  local arch_type="$(uname -m)"
  if [ "${arch_type}" == "x86_64" ]; then
    case "${platform_type}" in
      Linux|Darwin)
        export GOX_FLAGS="-arch amd64"
        export testResults="**/build/TEST*.xml"
        export artifacts="**/build/TEST*.out"
        ;;
      MINGW*)
        export GOX_FLAGS="-arch 386"
        export testResults="**\\build\\TEST*.xml"
        export artifacts="**\\build\\TEST*.out"
        ;;
    esac
  elif [[ "${arch_type}" == "aarch64" || "${arch_type}" == "arm64" ]]; then
    export GOX_FLAGS="-arch arm"
    export testResults="**/build/TEST*.xml"
    export artifacts="**/build/TEST*.out"
  else
    echo "Unsupported OS"
  fi
}

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
  local changelist=()
  local changed_files=$(git diff --name-only HEAD@{1} HEAD)
  if [ -z "$changed_files" ] || grep -qE "$(IFS=\|; echo "${patterns[*]}")" <<< "$changed_files"; then
    return 0
  fi
  return 1
}

withModule() {
  local module_path=$1
  if [[ "$module_path" == *"x-pack/"* ]]; then
    local pattern=("$XPACK_MODULE_PATTERN")
  else
    local pattern=("$OSS_MODULE_PATTERN")
  fi
  local module_name="${module_path#*module/}"
  local module_path_transformed=$(echo "$module_path" | sed 's/\//\\\//g')
  local module_path_exclussion="((?!^${module_path_transformed}\\/).)*\$"
  local exclude=("^(${module_path_transformed}|((?!\\/module\\/).)*\$|.*\\.asciidoc|.*\\.png)")
  if are_paths_changed "${pattern[@]}" && ! are_changed_only_paths "${exclude[@]}"; then
    MODULE="${module_name}"
  else
    MODULE=""
  fi
  echo "MODULE=$MODULE"
  export MODULE
}

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
  exportVars
  export RACE_DETECTOR="true"
  export TEST_COVERAGE="true"
  export DOCKER_PULL="0"
fi

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
  exportVars
  export RACE_DETECTOR="true"
  export TEST_COVERAGE="true"
  export DOCKER_PULL="0"
fi

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" ]]; then
  withModule "x-pack/metricbeat/module/aws"
fi
