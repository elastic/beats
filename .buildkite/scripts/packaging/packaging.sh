#!/usr/bin/env bash

set -ueo pipefail

BEAT_DIR=${1:-""}

if [ -z "$BEAT_DIR" ]; then
    echo "Error: Beat directory must be specified."
    exit 1
fi

docker run --privileged --rm tonistiigi/binfmt:master --install all

cd $BEAT_DIR
mage package
