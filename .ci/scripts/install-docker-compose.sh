#!/usr/bin/env bash
set -exo pipefail

function fetch() {
    if [ -e /usr/local/bin/bash_standard_lib.sh ] ; then
        # shellcheck disable=SC1091
        source /usr/local/bin/bash_standard_lib.sh
        retry 3 "$@"
    else
        "$@"
    fi
}

MSG="parameter missing."
DOCKER_COMPOSE_VERSION=${DOCKER_COMPOSE_VERSION:?$MSG}
HOME=${HOME:?$MSG}
DC_CMD="${HOME}/bin/docker-compose"

mkdir -p "${HOME}/bin"

retryCommand curl -sSLo "${DC_CMD}" "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)"
chmod +x "${DC_CMD}"
