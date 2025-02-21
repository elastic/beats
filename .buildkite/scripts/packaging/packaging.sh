#!/usr/bin/env bash

set -ueo pipefail

BEAT_DIR=${1:?-"Error: Beat directory must be specified."}

docker run --privileged --rm tonistiigi/binfmt:master --install all

cd $BEAT_DIR
mage package
