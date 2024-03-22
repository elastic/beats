#!/usr/bin/env bash

set -euo pipefail

echo "+++ Build Agent artifacts"

SNAPSHOT=""

BEAT_VERSION=$(grep -oE '[0-9]+\.[0-9]+\.[0-9]+(\-[a-zA-Z]+[0-9]+)?' "libbeat/version/version.go")
BEAT_VERSION_FULL=$BEAT_VERSION
if [ "$DRA_WORKFLOW" == "snapshot" ]; then
    SNAPSHOT="true"
    BEAT_VERSION_FULL="${BEAT_VERSION}-SNAPSHOT"
fi

PLATFORMS="darwin/arm64,darwin/amd64,linux/amd64,linux/arm64,windows/amd64"

PLATFORMS=$PLATFORMS SNAPSHOT=$SNAPSHOT mage -d x-pack/elastic-agent packageAgentCore
chmod 664 x-pack/elastic-agent/build/distributions/*

echo  "+++ Generate dependencies report"
./dev-tools/dependencies-report
mkdir -p x-pack/elastic-agent/build/distributions/reports
mv dependencies.csv "x-pack/elastic-agent/build/distributions/reports/dependencies-${BEAT_VERSION_FULL}.csv"
