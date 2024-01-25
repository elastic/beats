#!/bin/bash
set -euo pipefail

WORKSPACE=${WORKSPACE:-"$(pwd)"}
BIN="${WORKSPACE}/bin"
platform_type="$(uname)"
platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
arch_type="$(uname -m)"

DEBIAN_FRONTEND="noninteractive"
sudo mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null

with_docker_compose() {
  local version=$1
  echo "Setting up the Docker-compose environment..."
  create_workspace
  retry 5 curl -sSL -o ${BIN}/docker-compose "https://github.com/docker/compose/releases/download/${version}/docker-compose-${platform_type_lowercase}-${arch_type}"
  chmod +x ${BIN}/docker-compose
  export PATH="${PATH}:${BIN}"
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
  go mod download
}

with_python() {
  if [ "${platform_type}" == "Linux" ]; then
    sudo apt-get update
    sudo apt-get install -y python3-pip python3-venv libsystemd-dev libpcap-dev
  elif [ "${platform_type}" == "Darwin" ]; then
    brew update
    pip3 install virtualenv libpcap
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

config_git() {
  if [ -z "$(git config --get user.email)" ]; then
    git config --global user.email "beatsmachine@users.noreply.github.com"
    git config --global user.name "beatsmachine"
  fi
}


echo "--- Env preparation"

if command -v docker-compose &> /dev/null
then
  set +e
  FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version | awk '{print $3}' | sed s/\,//)
  echo "Found docker-compose version: $FOUND_DOCKER_COMPOSE_VERSION"
  if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]; then
    echo "Versions match. No need to install docker-compose. Exiting."
  else
    echo "Versions don't match. Need to install the correct version of docker-compose."
    with_docker_compose "${DOCKER_COMPOSE_VERSION}"
  fi
  set -e
else
  with_docker_compose "${DOCKER_COMPOSE_VERSION}"
fi

add_bin_path
with_go "${GO_VERSION}"
with_mage
with_python
config_git
mage dumpVariables
