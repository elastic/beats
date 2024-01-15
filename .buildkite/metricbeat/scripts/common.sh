#!/bin/bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}

create_workspace() {
  if [[ ! -d "${WORKSPACE}/bin" ]]; then
    mkdir -p "${WORKSPACE}/bin"
  fi
}

add_bin_path() {
  echo "Adding PATH to the environment variables..."
  create_workspace
  export PATH="${PATH}:${WORKSPACE}/bin"
}


# with_yq() {
#   pip install yq
# }

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
  go_version=$1
  url=$(get_gvm_link "${SETUP_GVM_VERSION}")
  create_workspace
  retry 5 curl -sL -o "${WORKSPACE}/bin/gvm" "${url}"
  chmod +x "${WORKSPACE}/bin/gvm"
  ls -l ${WORKSPACE}/bin/
  eval "$(gvm $go_version)"
  go_path="$(go env GOPATH):$(go env GOPATH)/bin"
  export PATH="${PATH}:${go_path}"
  go version
}

with_python() {
  if [ "$(uname)" == "Linux" ]; then
    sudo apt-get update
    sudo apt-get install -y libsystemd-dev
    sudo apt install -y python3-pip
    sudo apt-get install -y python3-venv
  fi
}

get_gvm_link() {
  gvm_version=$1
  platform_type="$(uname)"
  platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
  arch_type="$(uname -m)"
  [[ ${arch_type} == "aarch64" ]] && arch_type="arm64"
  [[ ${arch_type} == "x86_64" ]] && arch_type="amd64"
  echo "https://github.com/andrewkroh/gvm/releases/download/${gvm_version}/gvm-${platform_type_lowercase}-${arch_type}"
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

are_files_changed() {
  changeset=$1
  if git diff --name-only HEAD@{1} HEAD | grep -qE "$changeset"; then
    return 0;
  else
    echo "WARN! No files changed in $changeset"
    return 1;
  fi
}
