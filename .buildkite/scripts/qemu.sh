#!/usr/bin/env bash
#
# It enables multi-arch builds to avoid the exec format errors when 
# attempting to build arm64 inside arm64 workers.
#
# For further details, see https://github.com/elastic/elastic-agent/pull/6948
# and https://github.com/elastic/golang-crossbuild/pull/507
#
set -euo pipefail

if [[ "$(uname -m)" == "aarch64" || "$(uname -m)" == "arm64" ]]; then
    echo "Skipping qemu installation on arm64 worker"
else
    BINFMT_IMAGE="tonistiigi/binfmt:qemu-v9.2.2"

    # Make sure to uninstall first to avoid conflicts
    docker run --privileged --rm "$BINFMT_IMAGE" --uninstall qemu-*
    docker run --privileged --rm "$BINFMT_IMAGE" --install all
fi
