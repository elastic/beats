#!/usr/bin/env bash
set -uo pipefail
source ".buildkite/x-pack/elastic-agent/scripts/steps/common.sh"

if test -z "${ManifestURL=:""}"; then
  echo "Missing variable ManifestURL, export it before use."
  exit 2
fi

VERSION="$(make get-version)"
echo "--- Packaging Elastic Agent"

mage -v -d x-pack/elastic-agent clean downloadManifest package ironbank fixDRADockerArtifacts
