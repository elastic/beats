#!/usr/bin/env bash

set -exuo pipefail

MSG="environment variable missing: DOCKER_COMPOSE_VERSION."
DOCKER_COMPOSE_VERSION=${DOCKER_COMPOSE_VERSION:?$MSG}
HOME=${HOME:?$MSG}

if command -v docker-compose
then
    echo "Found docker-compose. Checking version.."
    FOUND_DOCKER_COMPOSE_VERSION=$(docker-compose --version|awk '{print $3}'|sed s/\,//)
    if [ $FOUND_DOCKER_COMPOSE_VERSION == $DOCKER_COMPOSE_VERSION ]
    then
        echo "Versions match. No need to install docker-compose. Exiting."
        exit 0
    fi
fi

echo "UNMET DEP: Installing docker-compose"

DC_CMD="${HOME}/bin/docker-compose"

mkdir -p "${HOME}/bin"

if curl -sSLo "${DC_CMD}" "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" ; then
    chmod +x "${DC_CMD}"
else
    echo "Something bad with the download, let's delete the corrupted binary"
    if [ -e "${DC_CMD}" ] ; then
        rm "${DC_CMD}"
    fi
    exit 1
fi
