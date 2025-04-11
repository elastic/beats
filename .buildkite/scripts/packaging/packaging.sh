#!/usr/bin/env bash
#
# Centralise the mage package for a given beat in Buildkite.
#

set -ueo pipefail

BEAT_DIR=${1:?-"Error: Beat directory must be specified."}

# shellcheck source=/dev/null
source .buildkite/scripts/qemu.sh

cd $BEAT_DIR
mage package
