#!/usr/bin/env bash
set -uo pipefail
source ".buildkite/x-pack/elastic-agent/scripts/steps/setenv.sh"
source ".buildkite/x-pack/elastic-agent/scripts/steps/common.sh"

DIST_PATH="x-pack/elastic-agent/build/distributions"

set -x
if test -z "${MANIFEST_URL=:""}"; then
  echo "Missing variable MANIFEST_URL, export it before use."
  exit 2
fi

VERSION="$(make get-version)"
echo "--- Packaging Elastic Agent"

echo $MANIFEST_URL

export AGENT_DROP_PATH=build/elastic-agent-drop
mkdir -p $AGENT_DROP_PATH

mage -v -d x-pack/elastic-agent clean downloadManifest package ironbank fixDRADockerArtifacts

echo  "+++ Generate dependencies report"
BEAT_VERSION_FULL=$(curl -s -XGET "${MANIFEST_URL}" |jq '.version' -r )
bash dev-tools/dependencies-report
mkdir -p $DIST_PATH/reports
mv dependencies.csv "$DIST_PATH/reports/dependencies-${BEAT_VERSION_FULL}.csv"
