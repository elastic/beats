#!/bin/bash

set -euo pipefail

WORKSPACE="$(pwd)/bin"

create_workspace() {
    if [[ ! -d "${WORKSPACE}" ]]; then
    mkdir -p ${WORKSPACE}
    fi
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

with_go() {
    local go_version=$1
    local gvm_version=$2
    url=$(get_gvm_link "${gvm_version}")
    WORKSPACE=${WORKSPACE:-"$(pwd)"}
    mkdir -p "${WORKSPACE}/bin"
    export PATH="${PATH}:${WORKSPACE}/bin"
    retry 5 curl -L -o "${WORKSPACE}/bin/gvm" "${url}"
    chmod +x "${WORKSPACE}/bin/gvm"
    ls ${WORKSPACE}/bin/
    eval "$(gvm $go_version)"
    go_path="$(go env GOPATH):$(go env GOPATH)/bin"
    export PATH="${PATH}:${go_path}"
    go version
}

# for gvm link
get_gvm_link() {
    local gvm_version=$1
    platform_type="$(uname)"
    arch_type="$(uname -m)"
    [[ ${arch_type} == "aarch64" ]] && arch_type="arm64" # gvm do not have 'aarch64' name for archetecture type
    [[ ${arch_type} == "x86_64" ]] && arch_type="amd64"
    echo "https://github.com/andrewkroh/gvm/releases/download/${gvm_version}/gvm-${platform_type}-${arch_type}"
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
