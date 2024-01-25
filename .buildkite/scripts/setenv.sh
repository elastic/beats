#!/bin/bash
set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
GO_VERSION=$(cat .go-version)
echo "GO_VERSION: ${GO_VERSION}"

os_name=$(uname)

case "$os_name" in
  Darwin | Linux)
    export DOCKER_COMPOSE_VERSION
    export SETUP_GVM_VERSION
    export GO_VERSION
    ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT)
    # TODO: Add environment variables from other pipelines, at least from metricbeat
    ;;
  *)
    echo "Unsupported operating system"
    exit 1
    ;;
esac
