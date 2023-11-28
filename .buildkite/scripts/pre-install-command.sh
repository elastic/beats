#!/bin/bash
set -euo pipefail

source .buildkite/scripts/tooling.sh

add_bin_path(){
    mkdir -p "${WORKSPACE}/bin"
    export PATH="${WORKSPACE}/bin:${PATH}"
}

with_mage() {
    mkdir -p "${WORKSPACE}/bin"
    retry 5 curl -sL -o "${WORKSPACE}/bin/mage.tar.gz" "https://github.com/magefile/mage/releases/download/v${SETUP_MAGE_VERSION}/mage_${SETUP_MAGE_VERSION}_Linux-64bit.tar.gz"

    tar -xvf "${WORKSPACE}/bin/mage.tar.gz" -C "${WORKSPACE}/bin"
    chmod +x "${WORKSPACE}/bin/mage"
    mage --version
}

with_go_junit_report() {
    go install github.com/jstemmer/go-junit-report/v2@latest
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

# Required env variables:
#   WORKSPACE
#   SETUP_MAGE_VERSION
WORKSPACE=${WORKSPACE:-"$(pwd)"}
