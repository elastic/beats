#!/usr/bin/env bash

set -euo pipefail
export REPO="beats"
export TMP_FOLDER="tmp.${REPO}"
export DOCKER_REGISTRY="docker.elastic.co"
export SETUP_GVM_VERSION="v0.5.1"
export DOCKER_COMPOSE_VERSION="1.21.0"
export DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
export SETUP_WIN_PYTHON_VERSION="3.11.0"
export NMAP_WIN_VERSION="7.12"           # Earlier versions of NMap provide WinPcap (the winpcap packages don't install nicely because they pop-up a UI)
export ASDF_MAGE_VERSION="1.15.0"
export PACKAGING_PLATFORMS="+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
export PACKAGING_ARM_PLATFORMS="linux/arm64"
export ASDF_TERRAFORM_VERSION="1.0.2"
export ASDF_NODEJS_VERSION="18.17.1"
export AWS_REGION="eu-central-1"
export NODEJS_VERSION="18.17.1"             #TODO remove after tests of the agent with nodeJS
export IMAGE_UBUNTU_X86_64="family/platform-ingest-beats-ubuntu-2204"
export IMAGE_UBUNTU_ARM_64="platform-ingest-beats-ubuntu-2204-aarch64"
export DEFAULT_UBUNTU_X86_64_IMAGE="family/core-ubuntu-2204"
export IMAGE_RHEL9_X86_64="family/platform-ingest-beats-rhel-9"
export IMAGE_WIN_10="family/platform-ingest-beats-windows-10"
export IMAGE_WIN_11="family/platform-ingest-beats-windows-11"
export IMAGE_WIN_2016="family/platform-ingest-beats-windows-2016"
export IMAGE_WIN_2019="family/platform-ingest-beats-windows-2019"
export IMAGE_WIN_2022="family/platform-ingest-beats-windows-2022"
export IMAGE_MACOS_X86_64="generic-13-ventura-x64"
export IMAGE_MACOS_ARM="generic-13-ventura-arm"
export GCP_DEFAULT_MACHINE_TYPE="c2d-highcpu-8"
export GCP_HI_PERF_MACHINE_TYPE="c2d-highcpu-16"
export GCP_WIN_MACHINE_TYPE="n2-standard-8"
export AWS_ARM_INSTANCE_TYPE="t4g.xlarge"
export WORKSPACE=${WORKSPACE:-"$(pwd)"}

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
        export MAGEFILE_CACHE="$WORKSPACE/$BEATS_PROJECT_NAME/.magefile"
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

if [[ "$BUILDKITE_STEP_KEY" == "xpack-winlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-metricbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-dockerlogbeat-pipeline" || "$BUILDKITE_STEP_KEY" == "xpack-filebeat-pipeline" || "$BUILDKITE_STEP_KEY" == "metricbeat-pipeline" || "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-heartbeat" ]]; then
  source .buildkite/scripts/common.sh
  if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-heartbeat" ]]; then
    export ELASTIC_SYNTHETICS_CAPABLE=true
  else
    # Set the MODULE env variable if possible, it should be defined before generating pipeline's steps. It is used in multiple pipelines.
    defineModuleFromTheChangeSet "${BEATS_PROJECT_NAME}"
  fi
fi

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-heartbeat" ]]; then
  # Set the MODULE env variable if possible, it should be defined before generating pipeline's steps. It is used in multiple pipelines.
  source .buildkite/scripts/common.sh
  defineModuleFromTheChangeSet "${BEATS_PROJECT_NAME}"
fi
