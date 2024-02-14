#!/usr/bin/env bash
set -uo pipefail
source ".buildkite/x-pack/elastic-agent/scripts/steps/common.sh"

VERSION="$(make get-version)"
echo "--- Packaging Elastic Agent"

mage -v -d x-pack/elastic-agent package

