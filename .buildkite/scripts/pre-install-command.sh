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

# Required env variables:
#   WORKSPACE
#   SETUP_MAGE_VERSION
WORKSPACE=${WORKSPACE:-"$(pwd)"}
