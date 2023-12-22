#!/bin/bash
set -euo pipefail

with_virtualenv() {
    export PATH=$PATH:/root/.local/bin
    python3 -m pip install --user virtualenv
}

with_yq() {
    pip install yq
}

with_mage() {
    mkdir -p "${WORKSPACE}/bin"
    retry 5 curl -sL -o "${WORKSPACE}/bin/mage.tar.gz" "https://github.com/magefile/mage/releases/download/v${SETUP_MAGE_VERSION}/mage_${SETUP_MAGE_VERSION}_Linux-64bit.tar.gz"

    tar -xvf "${WORKSPACE}/bin/mage.tar.gz" -C "${WORKSPACE}/bin"
    chmod +x "${WORKSPACE}/bin/mage"
    mage --version
}

with_gh() {
    # GitHub CLI linux amd64
    curl -L "https://github.com/cli/cli/releases/download/v${GITHUB_CLI}/gh_${GITHUB_CLI}_linux_amd64.tar.gz" --output ./gh.tar.gz
    mkdir gh
    tar -xf gh.tar.gz -C ./gh --strip-components=1
    chmod +x ./gh/bin/gh
    export PATH=$PATH:$(pwd)/gh/bin/
    gh --version
}

with_go() {
    go_version=$1
    url=$(get_gvm_link "${SETUP_GVM_VERSION}")
    WORKSPACE=${WORKSPACE:-"$(pwd)"}
    mkdir -p "${WORKSPACE}/bin"
    export PATH="${PATH}:${WORKSPACE}/bin"
    retry 5 curl -L -o "${WORKSPACE}/bin/gvm" "${url}"
    chmod +x "${WORKSPACE}/bin/gvm"
    ls ${WORKSPACE}/bin/ -l
    eval "$(gvm $go_version)"
    go_path="$(go env GOPATH):$(go env GOPATH)/bin"
    export PATH="${PATH}:${go_path}"
    go version
}

# for gvm link
get_gvm_link() {
    gvm_version=$1
    platform_type="$(uname)"
    platform_type_lowercase="${platform_type,,}"
    arch_type="$(uname -m)"
    [[ ${arch_type} == "aarch64" ]] && arch_type="arm64" # gvm do not have 'aarch64' name for archetecture type
    [[ ${arch_type} == "x86_64" ]] && arch_type="amd64"
    echo "https://github.com/andrewkroh/gvm/releases/download/${gvm_version}/gvm-${platform_type_lowercase}-${arch_type}"
}

# Required env variables:
#   WORKSPACE
WORKSPACE=${WORKSPACE:-"$(pwd)"}
