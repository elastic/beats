#!/usr/bin/env bash

set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_COMPOSE_VERSION_AARCH64="v2.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
NMAP_WIN_VERSION="7.12"           # Earlier versions of NMap provide WinPcap (the winpcap packages don't install nicely because they pop-up a UI)
GO_VERSION=$(cat .go-version)
ASDF_MAGE_VERSION="1.15.0"

export SETUP_GVM_VERSION
export DOCKER_COMPOSE_VERSION
export DOCKER_COMPOSE_VERSION_AARCH64
export SETUP_WIN_PYTHON_VERSION
export NMAP_WIN_VERSION
export GO_VERSION
export ASDF_MAGE_VERSION

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


if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-metricbeat" ]]; then
  exportVars
  export RACE_DETECTOR="true"
  export TEST_COVERAGE="true"
  export DOCKER_PULL="0"
fi
