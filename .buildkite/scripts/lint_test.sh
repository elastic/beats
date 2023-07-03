#!/bin/bash

set -euo pipefail

WORKSPACE="$(pwd)/bin"

create_workspace() {
    if [[ ! -d "${WORKSPACE}" ]]; then
    mkdir -p ${WORKSPACE}
    fi
}

with_go() {
    echo "Setting up the Go environment..."
    retry 5 curl -sL -o ${WORKSPACE}/gvm "https://github.com/andrewkroh/gvm/releases/download/${SETUP_GVM_VERSION}/gvm-linux-amd64"
    chmod +x ${WORKSPACE}/gvm
    eval "$(gvm $(cat .go-version))"
    go version
    which go
    export PATH="${PATH}:$(go env GOPATH):$(go env GOPATH)/bin"
    export GO_VERSION="$(go version)"
}

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

create_workspace

with_go

install_go_dependencies

mage -v check
