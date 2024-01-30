#!/bin/bash

set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
GO_VERSION=$(cat .go-version)
ALLOW_EXTENDED_TESTS=${ALLOW_EXTENDED_TESTS:-false}
ALLOW_MANDATORY_TESTS=${ALLOW_MANDATORY_TESTS:-false}
ALLOW_MACOS_TESTS=${ALLOW_MACOS_TESTS:-false}
ALLOW_EXTENDED_WIN_TESTS=${ALLOW_EXTENDED_WIN_TESTS:-false}
ALLOW_PACKAGING=${ALLOW_PACKAGING:-false}
GITHUB_PR_TRIGGER_COMMENT=${GITHUB_PR_TRIGGER_COMMENT:-""}
ONLY_DOCS=${ONLY_DOCS:-"true"}
BEATS_PROJECT_NAME="metricbeat"
UI_MACOS_TESTS="$(buildkite-agent meta-data get UI_MACOS_TESTS --default ${UI_MACOS_TESTS:-"false"})"
runAllStages="$(buildkite-agent meta-data get runAllStages --default ${runAllStages:-"false"})"
metricbeat_changeset=(
  "^metricbeat/.*"
  "^go.mod"
  "^pytest.ini"
  "^dev-tools/.*"
  "^libbeat/.*"
  "^testing/.*"
  )
oss_changeset=(
  "^go.mod"
  "^pytest.ini"
  "^dev-tools/.*"
  "^libbeat/.*"
  "^testing/.*"
)
ci_changeset=(
  "^.buildkite/.*"
)
