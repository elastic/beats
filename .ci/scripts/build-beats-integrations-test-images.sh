#!/usr/bin/env bash
set -exo pipefail

#
# Install the given go version using the gimme script.
#
# Parameters:
#   - GO_VERSION - that's the version which will be installed and enabled.
#   - METRICBEAT_DIR - that's the location of the metricbeat directory.
#

readonly GO_VERSION="${1?Please define the Go version to be used}"
readonly METRICBEAT_DIR="${2?Please define the location of the Metricbeat directory}"

function build_test_images() {
    local metricbeatDir="${1}"

    cd "${metricbeatDir}"
    mage compose:buildSupportedVersions
}

function install_go() {
    local goVersion="${1}"

    # Install Go using the same travis approach
    echo "Installing ${goVersion} with gimme."
    eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=${goVersion} bash)"
}

function install_mage() {
    local metricbeatDir="${1}"

    cd "${metricbeatDir}"
    make mage
}

function push_test_images() {
    local metricbeatDir="${1}"

    cd "${metricbeatDir}"
    mage compose:pushSupportedVersions
}

function main() {
    install_go "${GO_VERSION}"
    install_mage "${METRICBEAT_DIR}"

    build_test_images "${METRICBEAT_DIR}"
    push_test_images "${METRICBEAT_DIR}"
}

main "$@"
