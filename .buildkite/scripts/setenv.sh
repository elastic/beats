#!/usr/bin/env bash

set -euo pipefail
REPO="beats"
TMP_FOLDER="tmp.${REPO}"
DOCKER_REGISTRY="docker.elastic.co"
SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
NMAP_WIN_VERSION="7.12"           # Earlier versions of NMap provide WinPcap (the winpcap packages don't install nicely because they pop-up a UI)
GO_VERSION=$(cat .go-version)
ASDF_MAGE_VERSION="1.15.0"
PACKAGING_PLATFORMS="+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"
PACKAGING_ARM_PLATFORMS="linux/arm64"
ASDF_TERRAFORM_VERSION="1.0.2"
AWS_REGION="eu-central-1"
NODEJS_VERSION="18.17.1"
IMAGE_UBUNTU_X86_64="family/platform-ingest-beats-ubuntu-2204"
IMAGE_UBUNTU_ARM_64="platform-ingest-beats-ubuntu-2204-aarch64"
DEFAULT_UBUNTU_X86_64_IMAGE="family/core-ubuntu-2204"
IMAGE_RHEL9_X86_64="family/platform-ingest-beats-rhel-9"
IMAGE_WIN_10="family/general-windows-10"
IMAGE_WIN_11="family/general-windows-11"
IMAGE_WIN_2016="family/core-windows-2016"
IMAGE_WIN_2019="family/core-windows-2019"
IMAGE_WIN_2022="family/core-windows-2022"
IMAGE_MACOS_X86_64="generic-13-ventura-x64"
GCP_DEFAULT_MACHINE_TYPE="c2d-highcpu-8"
GCP_HI_PERF_MACHINE_TYPE="c2d-highcpu-16"
GCP_WIN_MACHINE_TYPE="n2-standard-8"
AWS_ARM_INSTANCE_TYPE="t4g.xlarge"
BEATS_PROJECT_NAME="x-pack/packetbeat"

export SETUP_GVM_VERSION
export DOCKER_COMPOSE_VERSION
export DOCKER_COMPOSE_VERSION_AARCH64
export SETUP_WIN_PYTHON_VERSION
export NMAP_WIN_VERSION
export GO_VERSION
export ASDF_MAGE_VERSION
export PACKAGING_PLATFORMS
export PACKAGING_ARM_PLATFORMS
export REPO
export TMP_FOLDER
export DOCKER_REGISTRY
export ASDF_TERRAFORM_VERSION
export AWS_REGION
export NODEJS_VERSION

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
