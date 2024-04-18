#!/usr/bin/env bash

set -euo pipefail

export REPO="beats"
export DOCKER_REGISTRY="docker.elastic.co"
export SETUP_GVM_VERSION="v0.5.1"
export DOCKER_COMPOSE_VERSION="1.21.0"
export DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
export ASDF_NODEJS_VERSION="18.17.1"
export AWS_REGION="eu-central-1"

WORKSPACE=${WORKSPACE:-"$(pwd)"}
export WORKSPACE
GO_VERSION=$(cat .go-version)
export GO_VERSION


exportVars() {
  local platform_type="$(uname)"
  local arch_type="$(uname -m)"
  if [ "${arch_type}" == "x86_64" ]; then
    case "${platform_type}" in
      Linux|Darwin)
        export GOX_FLAGS="-arch amd64"
        export testResults="**/build/TEST*.xml"
        export artifacts="**/build/TEST*.out"
        ;;
      MINGW*)
        export GOX_FLAGS="-arch 386"
        export testResults="**\\build\\TEST*.xml"
        export artifacts="**\\build\\TEST*.out"
        ;;
    esac
  elif [[ "${arch_type}" == "aarch64" || "${arch_type}" == "arm64" ]]; then
    export GOX_FLAGS="-arch arm"
    export testResults="**/build/TEST*.xml"
    export artifacts="**/build/TEST*.out"
  else
    echo "Unsupported OS"
  fi
}

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-metricbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-winlogbeat" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-auditbeat" ]]; then
  exportVars
  export RACE_DETECTOR="true"
  export TEST_COVERAGE="true"
  export DOCKER_PULL="0"
  export TEST_TAGS="${TEST_TAGS:+$TEST_TAGS,}oracle"
fi

<<<<<<< HEAD
if [[ "$BUILDKITE_STEP_KEY" == "xpack-winlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-metricbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-dockerlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-filebeat-pipeline" || "$BUILDKITE_STEP_KEY" == "metricbeat-pipeline" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-heartbeat" ]]; then
=======
if [[ "$BUILDKITE_STEP_KEY" == "xpack-winlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-metricbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-dockerlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "metricbeat-pipeline" ]]; then
>>>>>>> 0d3a8b68d9 ([CI] x-pack/heartbeat monorepo (#38845))
  source .buildkite/scripts/common.sh
  # Set the MODULE env variable if possible, it should be defined before generating pipeline's steps. It is used in multiple pipelines.
  defineModuleFromTheChangeSet "${BEATS_PROJECT_NAME}"
fi
