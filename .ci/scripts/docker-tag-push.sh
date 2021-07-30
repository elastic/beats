#!/usr/bin/env bash
set -exuo pipefail
MSG="parameter missing."
SOURCE_IMAGE=${1:?$MSG}
TARGET_IMAGE=${2:?$MSG}

if docker image inspect "${SOURCE_IMAGE}" &> /dev/null ; then
    docker tag "${SOURCE_IMAGE}" "${TARGET_IMAGE}"
    docker push "${TARGET_IMAGE}"
else
    echo "docker image ${SOURCE_IMAGE} does not exist"
fi
