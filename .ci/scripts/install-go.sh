#!/usr/bin/env bash
set -exo pipefail

function fetch() {
    ## Load CI functions
    # shellcheck disable=SC1091
    if [ -e /usr/local/bin/bash_standard_lib.sh ] ; then
        source /usr/local/bin/bash_standard_lib.sh
    fi
    if [ -n "${JENKINS_URL}" ] ; then
        retry 2 "$@"
    else
        "$@"
    fi
}

MSG="parameter missing."
GO_VERSION=${GO_VERSION:?$MSG}
PROPERTIES_FILE=${PROPERTIES_FILE:-"go_env.properties"}
HOME=${HOME:?$MSG}
ARCH=$(uname -s| tr '[:upper:]' '[:lower:]')
GVM_CMD="${HOME}/bin/gvm"

mkdir -p "${HOME}/bin"

fetch -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-${ARCH}-amd64"
chmod +x "${GVM_CMD}"

gvm ${GO_VERSION}|cut -d ' ' -f 2|tr -d '\"' > ${PROPERTIES_FILE}

eval $(gvm ${GO_VERSION})
