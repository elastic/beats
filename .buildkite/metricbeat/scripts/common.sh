#!/bin/bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}
BIN="${WORKSPACE}/bin"
platform_type="$(uname)"
platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
arch_type="$(uname -m)"
pipeline_name="metricbeat"

with_docker_compose() {
  local version=$1
  echo "Setting up the Docker-compose environment..."
  create_workspace
  retry 5 curl -sSL -o ${BIN}/docker-compose "https://github.com/docker/compose/releases/download/${version}/docker-compose-${platform_type_lowercase}-${arch_type}"
  chmod +x ${BIN}/docker-compose
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
  export PATH="${PATH}:${BIN}"
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
  local version=$1
  if [ "${platform_type}" == "Linux" ]; then
    sudo apt-get update
    sudo apt-get install -y python3-venv python3-pip libsystemd-dev pytest pluggy
  elif [ "${platform_type}" == "Darwin" ]; then
    brew update
    brew install pyenv
    pyenv install "${version}"
    pyenv global "${version}"
    python --version
    pip3 install --upgrade pip
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

are_files_changed() {
  changeset=$1
  if git diff --name-only HEAD@{1} HEAD | grep -qE "$changeset"; then
    return 0;
  else
    echo "WARN! No files changed in $changeset"
    return 1;
  fi
}

echo "--- Env preparation"

if command -v docker-compose &> /dev/null
then
  set +e
  echo "Found docker-compose. Checking version.."
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version|awk '{print $3}'|sed s/\,//)
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  else
    echo "Versions don't match. Need to install the correct version of docker-compose."
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
  set -e
fi

add_bin_path
with_go "${GO_VERSION}"
with_mage
with_python "${SETUP_PYTHON_VERSION}"
