#!/bin/bash
set -euo pipefail

SETUP_GVM_VERSION="v0.5.1"
DOCKER_COMPOSE_VERSION="1.21.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"
GO_VERSION=$(cat .go-version)
echo "GO_VERSION: ${GO_VERSION}"
os_type=$(uname)

case "$os_type" in
  Darwin | Linux)
    export DOCKER_COMPOSE_VERSION
    export SETUP_GVM_VERSION
    export GO_VERSION
    ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT)
    export SETUP_WIN_PYTHON_VERSION
    ;;
  *)
    echo "Unsupported operating system"
    exit 1
    ;;
esac
