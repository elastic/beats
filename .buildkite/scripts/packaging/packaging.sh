#!/usr/bin/env bash
#
# Centralise the mage package for a given beat in Buildkite.
# It enables multi-arch builds to avoid the exec format errors when 
# attempting to build arm64 inside arm64 workers.
# For further details, see https://github.com/elastic/elastic-agent/pull/6948
# and https://github.com/elastic/golang-crossbuild/pull/507
#

set -ueo pipefail


BEAT_DIR=${1:?-"Error: Beat directory must be specified."}

# NOTE: Pinned to QEMU v8.x since v9.x breaks compilation of amd64 binaries in arm64.
# See: https://gitlab.com/qemu-project/qemu/-/issues/2560
BINFMT_IMAGE="tonistiigi/binfmt:qemu-v8.1.5"
# Make sure to uninstall first to avoid conflicts
docker run --privileged --rm "$BINFMT_IMAGE" --uninstall qemu-*
docker run --privileged --rm "$BINFMT_IMAGE" --install all

cd $BEAT_DIR
mage package
