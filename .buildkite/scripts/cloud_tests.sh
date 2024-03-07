#!/usr/bin/env bash

source .buildkite/scripts/install_tools.sh

set -euo pipefail

echo "--- Run Cloud Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}" > /dev/null

MODULE=aws mage build test

popd > /dev/null

echo "---Terraform Cleanup"
.ci/scripts/terraform-cleanup.sh "${MODULE_DIR}"              #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK

echo "---Docker Compose Cleanup"
docker-compose -f .ci/jobs/docker-compose.yml down -v         #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK
