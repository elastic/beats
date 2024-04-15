#!/usr/bin/env bash

set -euo pipefail

GO_VERSION=$(cat .go-version)
SETUP_GVM_VERSION="v0.5.1"
PLATFORM_TYPE_LOWERCASE=$(uname | tr '[:upper:]' '[:lower:]')

export BIN=${WORKSPACE:-$PWD}/bin

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

define_arch() {
  local platform_type="$(uname)"
  local arch_type="$(uname -m)"
  if [ "${arch_type}" == "x86_64" ]; then
        export GOX_FLAGS="-arch amd64"
        go_arch_type="amd64"
  elif [[ "${arch_type}" == "aarch64" || "${arch_type}" == "arm64" ]]; then
    export GOX_FLAGS="-arch arm"
    go_arch_type="arm64"
  else
    echo "+++ Unsupported OS archictecture; uname: $platform_type and uname -m: $arch_type"
    exit 1
  fi
}

create_workspace() {
  if [[ ! -d "${BIN}" ]]; then
    mkdir -p "${BIN}"
  fi
}

with_docker_compose() {
  local version=$1
  echo "Setting up the Docker-compose environment..."
  create_workspace
  retry 3 curl -sSL -o ${BIN}/docker-compose "https://github.com/docker/compose/releases/download/${version}/docker-compose-${PLATFORM_TYPE_LOWERCASE}-${arch_type}"
  chmod +x ${BIN}/docker-compose
  export PATH="${BIN}:${PATH}"
  docker-compose version
}

add_bin_path() {
  echo "Adding PATH to the environment variables..."
  create_workspace
  export PATH="${BIN}:${PATH}"
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
  echo "Download modules to local cache"
  retry 3 go mod download
}

with_go() {
  echo "Setting up the Go environment..."
  create_workspace
  define_arch
  retry 5 curl -sL -o "${BIN}/gvm" "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-${PLATFORM_TYPE_LOWERCASE}-${go_arch_type}"
  chmod +x "${BIN}/gvm"
  eval "$(gvm $GO_VERSION)"
  go version
  which go
  local go_path="$(go env GOPATH):$(go env GOPATH)/bin"
  export PATH="${go_path}:${PATH}"
}

with_python() {
    brew update
    pip3 install virtualenv
    ulimit -Sn 10000
}

config_git() {
  if [ -z "$(git config --get user.email)" ]; then
    git config --global user.email "beatsmachine@users.noreply.github.com"
    git config --global user.name "beatsmachine"
  fi
}

withNodeJSEnv() {
  local version=$1
  echo "~~~ Installing nvm and Node.js"
  curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
  export NVM_DIR="$HOME/.nvm"
  [ -s "$NVM_DIR/nvm.sh" ] && source "$NVM_DIR/nvm.sh"
  echo "Installing Node.js version: $version"
  nvm install "$version"
  # export PATH="${nvmPath}:${PATH}"
  nvm use "$version"
  node --version
  echo "~~~ Resuming commands"
}

installNodeJsDependencies() {
  echo "~~~ Installing Node.js packages"
  # needed for beats-xpack-heartbeat
  echo "Install @elastic/synthetics"
  npm i -g @elastic/synthetics
  echo "~~~ Resuming commands"
}

add_bin_path
with_go "${GO_VERSION}"
with_mage
with_python
config_git
