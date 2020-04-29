#!/usr/bin/env bash
set -exo pipefail

function retryCommand() {
    if [ -e /usr/local/bin/bash_standard_lib.sh ] ; then
        # shellcheck disable=SC1091
        source /usr/local/bin/bash_standard_lib.sh
        retry 3 "$@"
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

retryCommand curl -sSLo "${GVM_CMD}" "https://github.com/andrewkroh/gvm/releases/download/v0.2.2/gvm-${ARCH}-amd64"
chmod +x "${GVM_CMD}"

gvm ${GO_VERSION}|cut -d ' ' -f 2|tr -d '\"' > ${PROPERTIES_FILE}

eval $(gvm ${GO_VERSION})
