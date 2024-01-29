#!/bin/bash

set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
GO_VERSION=$(cat .go-version)
platform_type="$(uname)"
platform_type_lowercase=$(echo "$platform_type" | tr '[:upper:]' '[:lower:]')
arch_type="$(uname -m)"
ALLOW_EXTENDED_TESTS=${ALLOW_EXTENDED_TESTS:-false}
ALLOW_MANDATORY_TESTS=${ALLOW_MANDATORY_TESTS:-false}
ALLOW_MACOS_TESTS=${ALLOW_MACOS_TESTS:-false}
ALLOW_EXTENDED_WIN_TESTS=${ALLOW_EXTENDED_WIN_TESTS:-false}
ALLOW_PACKAGING=${ALLOW_PACKAGING:-false}
GITHUB_PR_TRIGGER_COMMENT=${GITHUB_PR_TRIGGER_COMMENT:-""}
ONLY_DOCS=${ONLY_DOCS:-"true"}
# GO_MOD_CHANGES=${GO_MOD_CHANGES:-"false"}
# PACKAGING_CHANGES=${PACKAGING_CHANGES:-"false"}
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
pipeline_name="metricbeat"

# case "$arch_type" in
#   Darwin | Linux)
#     export DOCKER_COMPOSE_VERSION
#     export SETUP_GVM_VERSION
#     export GO_VERSION
#     ;;
#   MINGW* | MSYS* | CYGWIN* | Windows_NT)
#     export SETUP_WIN_PYTHON_VERSION
#     ;;
#   *)
#     echo "Unsupported operating system"
#     exit 1
#     ;;
# esac
