#!/bin/bash

set -euo pipefail

# WORKSPACE="$(go env GOPATH)/bin"

# create_workspace() {
#     if [[ ! -d "${WORKSPACE}" ]]; then
#     mkdir -p ${WORKSPACE}
#     fi
# }

# with_go() {
#     echo "Setting up the Go environment..."
#     retry 5 curl -sL -o ${WORKSPACE}/gvm "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-linux-amd64"
#     chmod +x ${WORKSPACE}/gvm
#     eval "$(gvm $(cat .go-version))"
#     go version
#     which go
#     export PATH="${PATH}:$(go env GOPATH):$(go env GOPATH)/bin"
# }

install_go_dependencies() {
    local install_packages=(
            "github.com/magefile/mage"
            "github.com/elastic/go-licenser"
            "golang.org/x/tools/cmd/goimports"
            "github.com/jstemmer/go-junit-report"
            "gotest.tools/gotestsum"
    )
    for pkg in "${install_packages[@]}"; do
        go install "${pkg}@latest"
    done
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
