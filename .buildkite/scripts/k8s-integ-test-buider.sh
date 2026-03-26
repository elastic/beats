#!/usr/bin/env bash
set -ueo pipefail

# shellcheck source=/dev/null
source .buildkite/scripts/docker.sh

cd filebeat
mage package
