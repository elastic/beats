#!/usr/bin/env bash
set -exuo pipefail
MSG="parameter missing."
sourceName=${1:?$MSG}
targetName=${2:?$MSG}

if docker image inspect "${sourceName}" &> /dev/null ; then
    docker tag ${sourceName} ${targetName}
    docker push ${targetName}
else
    echo "docker image ${sourceName} does not exist"
fi
