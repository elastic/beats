#!/usr/bin/env bash

set -exuo pipefail

MSG="parameter missing."
DOCKER_COMPOSE_VERSION=${DOCKER_COMPOSE_VERSION:?$MSG}
HOME=${HOME:?$MSG}
DC_CMD="${HOME}/bin/docker-compose"

mkdir -p "${HOME}/bin"

curl -sSLo "${DC_CMD}" "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)"
chmod +x "${DC_CMD}"
