#!/usr/bin/env bash
set -exo pipefail

#
# Install the given go version using the gimme script.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#   - BEAT_BASE_DIR - that's the base directory of the Beat.
#

readonly GO_VERSION="${1?Please define the Go version to be used}"
readonly BEAT_BASE_DIR="${2?Please define the location of the Beat directory}"

function build_test_images() {
    local baseDir="${1}"

    cd "${baseDir}"
    mage compose:buildSupportedVersions
}

function install_go() {
    local goVersion="${1}"

    # Install Go using the same travis approach
    echo "Installing ${goVersion} with gimme."
    eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${goVersion} bash)"
}

function install_mage() {
    local baseDir="${1}"

    cd "${baseDir}"
    make mage
}

function push_test_images() {
    local baseDir="${1}"

    cd "${baseDir}"
    mage compose:pushSupportedVersions
}

function main() {
    install_go "${GO_VERSION}"
    install_mage "${BEAT_BASE_DIR}"

    build_test_images "${BEAT_BASE_DIR}"
    push_test_images "${BEAT_BASE_DIR}"
}

main "$@"
