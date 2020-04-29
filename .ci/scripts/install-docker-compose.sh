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
DOCKER_COMPOSE_VERSION=${DOCKER_COMPOSE_VERSION:?$MSG}
HOME=${HOME:?$MSG}
DC_CMD="${HOME}/bin/docker-compose"

mkdir -p "${HOME}/bin"

fetch curl -sSLo "${DC_CMD}" "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)"
chmod +x "${DC_CMD}"
