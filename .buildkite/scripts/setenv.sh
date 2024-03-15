#!/usr/bin/env bash

set -euo pipefail
REPO="beats"
TMP_FOLDER="tmp.${REPO}"
DOCKER_REGISTRY="docker.elastic.co"
SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
NMAP_WIN_VERSION="7.12"           # Earlier versions of NMap provide WinPcap (the winpcap packages don't install nicely because they pop-up a UI)
GO_VERSION=$(cat .go-version)
ASDF_MAGE_VERSION="1.15.0"
PACKAGING_PLATFORMS="+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
PACKAGING_ARM_PLATFORMS="linux/arm64"
ASDF_TERRAFORM_VERSION="1.0.2"
AWS_REGION="eu-central-1"

export SETUP_GVM_VERSION
export DOCKER_COMPOSE_VERSION
export DOCKER_COMPOSE_VERSION_AARCH64
export SETUP_WIN_PYTHON_VERSION
export NMAP_WIN_VERSION
export GO_VERSION
export ASDF_MAGE_VERSION
export PACKAGING_PLATFORMS
export PACKAGING_ARM_PLATFORMS
export REPO
export TMP_FOLDER
export DOCKER_REGISTRY
export ASDF_TERRAFORM_VERSION
export AWS_REGION

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
  # This method gathers the module name, if required, in order to run the ITs only if the changeset affects a specific module.
  # For such, it's required to look for changes under the module folder and exclude anything else such as asciidoc and png files.
  # This method defines and exports the MODULE variable with a particular module name or '' if changeset doesn't affect a specific module
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
  if [[ -z "$changed_modules" ]]; then # TODO: remove this condition and uncomment the line below when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
    export MODULE="aws"
  else
    export MODULE="${changed_modules}"  # TODO: remove this line and uncomment the line below when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
  # export MODULE="${changed_modules}"     # TODO: uncomment the line when the issue https://github.com/elastic/ingest-dev/issues/2993 is solved
  fi
}

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-winlogbeat" ]]; then
  exportVars
  export RACE_DETECTOR="true"
  export TEST_COVERAGE="true"
  export DOCKER_PULL="0"
  export TEST_TAGS="${TEST_TAGS:+$TEST_TAGS,}oracle"
fi
