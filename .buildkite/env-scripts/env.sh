#!/usr/bin/env bash

REPO="beats"
SETUP_GVM_VERSION="v0.5.1"
WORKSPACE="$(pwd)"
BIN="${WORKSPACE}/bin"
HW_TYPE="$(uname -m)"
PLATFORM_TYPE="$(uname)"
REPO="beats"
TMP_FOLDER="tmp.${REPO}"
DOCKER_REGISTRY="docker.elastic.co"

export SETUP_GVM_VERSION
export WORKSPACE
export BIN
export HW_TYPE
export PLATFORM_TYPE
export REPO
export TMP_FOLDER
export DOCKER_REGISTRY
