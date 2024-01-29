#!/bin/bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}
BIN="${WORKSPACE}/bin"
platform_type="$(uname)"
platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
arch_type="$(uname -m)"
pipeline_name="metricbeat"
DEBIAN_FRONTEND="noninteractive"
# sudo mkdir -p /etc/needrestart
# echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null
mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | tee -a /etc/needrestart/needrestart.conf > /dev/null

with_docker_compose() {
  local version=$1
  echo "Setting up the Docker-compose environment..."
  create_workspace
  retry 5 curl -sSL -o ${BIN}/docker-compose "https://github.com/docker/compose/releases/download/${version}/docker-compose-${platform_type_lowercase}-${arch_type}"
  chmod +x ${BIN}/docker-compose
  export PATH="${BIN}:${PATH}"
  docker-compose version
}

create_workspace() {
  if [[ ! -d "${BIN}" ]]; then
    mkdir -p "${BIN}"
  fi
}

add_bin_path() {
  echo "Adding PATH to the environment variables..."
  create_workspace
  export PATH="${BIN}:${PATH}"
}

check_platform_architeture() {
  case "${arch_type}" in
    "x86_64")
      arch_type="amd64"
      ;;
    "aarch64")
      arch_type="arm64"
      ;;
    "arm64")
      arch_type="arm64"
      ;;
    *)
    echo "The current platform/OS type is unsupported yet"
    ;;
  esac
}

with_mage() {
  local install_packages=(
    "github.com/magefile/mage"
    "github.com/elastic/go-licenser"
    "golang.org/x/tools/cmd/goimports"
    "github.com/jstemmer/go-junit-report"
    "gotest.tools/gotestsum"
  )
  create_workspace
  for pkg in "${install_packages[@]}"; do
    go install "${pkg}@latest"
  done
}

with_go() {
  echo "Setting up the Go environment..."
  create_workspace
  check_platform_architeture
  retry 5 curl -sL -o "${BIN}/gvm" "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-${platform_type_lowercase}-${arch_type}"
  chmod +x "${BIN}/gvm"
  eval "$(gvm $GO_VERSION)"
  go version
  which go
  local go_path="$(go env GOPATH):$(go env GOPATH)/bin"
  export PATH="${PATH}:${go_path}"
}

with_python() {
  if [ "${platform_type}" == "Linux" ]; then
    # sudo apt-get update
    # sudo apt-get install -y python3-pip python3-venv libsystemd-dev
    apt-get update
    apt-get install -y python3-pip python3-venv libsystemd-dev
  elif [ "${platform_type}" == "Darwin" ]; then
    brew update
    pip3 install virtualenv
    ulimit -Sn 10000
  fi
}

retry() {
  local retries=$1
  shift
  local count=0
  until "$@"; do
    exit=$?
    wait=$((2 ** count))
    count=$((count + 1))
    if [ $count -lt "$retries" ]; then
      >&2 echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      >&2 echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

are_paths_changed() {
  local patterns=("${@}")
  local changelist=()
  for pattern in "${patterns[@]}"; do
    changed_files=$(git diff --name-only HEAD@{1} HEAD | grep -E "$pattern")
    if [ -n "$changed_files" ]; then
      changelist+=("${changed_files}")
    fi
  done
  if [[ -n "${changelist[@]}" ]]; then
    echo "Files changed:"
    echo "${changelist[*]}"
    return 0
  else
    echo "No files changed within Metricbeat changeset"
    return 1
  fi
}

are_changed_only_paths() {
  local patterns=("${@}")
  local changelist=()
  local changed_files=$(git diff --name-only HEAD@{1} HEAD)
  if [ -z "$changed_files" ] || grep -qE "$(IFS=\|; echo "${patterns[*]}")" <<< "$changed_files"; then
    echo "All changes are within the specified patterns or there are no changes at all."
    return 0
  else
    echo "Changes include files outside the specified patterns."
    return 1
  fi
}
