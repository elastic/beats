#!/usr/bin/env bash

source .buildkite/env-scripts/util.sh

DOCS_CHANGESET="^.*\.(asciidoc|md)$
deploy/kubernetes/.*-kubernetes.yaml"
PACKAGING_CHANGESET="^dev-tools/packaging/
^.go-version"

REPO="beats"
SNAPSHOT="true"

ASDF_MAGE_VERSION="1.15.0"

# Docker & DockerHub
DOCKER_COMPOSE_VERSION="1.21.0"
WORKSPACE="$(pwd)"
BIN="${WORKSPACE}/bin"
HW_TYPE="$(uname -m)"
PLATFORM_TYPE="$(uname)"
REPO="beats"
TMP_FOLDER="tmp.${REPO}"
DOCKER_REGISTRY="docker.elastic.co"

export WORKSPACE
export BIN
export HW_TYPE
export PLATFORM_TYPE
export REPO
export TMP_FOLDER
export DOCKER_REGISTRY
export SNAPSHOT
export ASDF_MAGE_VERSION
export DOCKER_COMPOSE_VERSION
