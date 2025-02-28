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

cd $BEAT_DIR
mage package
