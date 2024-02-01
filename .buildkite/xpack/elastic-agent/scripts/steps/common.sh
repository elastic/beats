#!/bin/bash

set -euo pipefail

if [[ -z "${WORKSPACE-""}" ]]; then
    WORKSPACE=$(git rev-parse --show-toplevel)
    export WORKSPACE
fi

if [[ -z "${SETUP_MAGE_VERSION-""}" ]]; then
    SETUP_MAGE_VERSION="v0.5.0"
fi
if [[ -z "${SETUP_GVM_VERSION-""}" ]]; then
    SETUP_GVM_VERSION="1.14.0"
fi

getOSOptions() {
  case $(uname | tr '[:upper:]' '[:lower:]') in
    linux*)
      export AGENT_OS_NAME=linux
      ;;
    darwin*)
      export AGENT_OS_NAME=darwin
      ;;
    msys*)
      export AGENT_OS_NAME=windows
      ;;
    *)
      export AGENT_OS_NAME=notset
      ;;
  esac
  case $(uname -m | tr '[:upper:]' '[:lower:]') in
    aarch64*)
      export AGENT_OS_ARCH=arm64
      ;;
    arm64*)
      export AGENT_OS_ARCH=arm64
      ;;
    amd64*)
      export AGENT_OS_ARCH=amd64
      ;;
    x86_64*)
      export AGENT_OS_ARCH=amd64
      ;;
    *)
      export AGENT_OS_ARCH=notset
      ;;
  esac
}

# Wrapper function for executing mage
mage() {
    go version
    if ! [ -x "$(type -P mage | sed 's/mage is //g')" ];
    then
        echo "installing mage ${SETUP_MAGE_VERSION}"
        make mage
    fi
    pushd "$WORKSPACE"
    command "mage" "$@"
    ACTUAL_EXIT_CODE=$?
    popd
    return $ACTUAL_EXIT_CODE
}

# Wrapper function for executing go
go(){
    # Search for the go in the Path
    if ! [ -x "$(type -P go | sed 's/go is //g')" ];
    then
        getOSOptions
        echo "installing golang "${GO_VERSION}" for "${AGENT_OS_NAME}/${AGENT_OS_ARCH}" "
        local _bin="${WORKSPACE}/bin"
        mkdir -p "${_bin}"
        retry 5 curl -sL -o "${_bin}/gvm" "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-${AGENT_OS_NAME}-${AGENT_OS_ARCH}"
        chmod +x "${_bin}/gvm"
        eval "$(command "${_bin}/gvm" "${GO_VERSION}" )"
        export GOPATH=$(command go env GOPATH)
        export PATH="${PATH}:${GOPATH}/bin"
    fi
    pushd "$WORKSPACE"
    command go "$@"
    ACTUAL_EXIT_CODE=$?
    popd
    return $ACTUAL_EXIT_CODE
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
