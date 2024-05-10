#!/usr/bin/env bash

source .buildkite/env-scripts/util.sh

DOCS_CHANGESET="^.*\.(asciidoc|md)$
deploy/kubernetes/.*-kubernetes.yaml"
PACKAGING_CHANGESET="^dev-tools/packaging/
^.go-version"

REPO="beats"
WORKSPACE="$(pwd)"
BIN="${WORKSPACE}/bin"
HW_TYPE="$(uname -m)"
PLATFORM_TYPE="$(uname)"
SNAPSHOT="true"
PYTEST_ADDOPTS=""

SETUP_GVM_VERSION="v0.5.1"
ASDF_MAGE_VERSION="1.14.0"
SETUP_WIN_PYTHON_VERSION="3.11.0"

# Docker & DockerHub
DOCKER_COMPOSE_VERSION="1.21.0"
DOCKER_REGISTRY="docker.elastic.co"

ONLY_DOCS=$(changeset_applies "$DOCS_CHANGESET")
PACKAGING_CHANGES=$(changeset_applies "$PACKAGING_CHANGESET")
GO_MOD_CHANGES=$(changeset_applies "^go.mod")

KIND_VERSION="v0.20.0"
KUBECONFIG="${WORKSPACE}/kubecfg"

export WORKSPACE
export BIN
export HW_TYPE
export PLATFORM_TYPE
export SNAPSHOT
export PYTEST_ADDOPTS

export SETUP_GVM_VERSION
export ASDF_MAGE_VERSION
export SETUP_WIN_PYTHON_VERSION

export DOCKER_COMPOSE_VERSION
export DOCKER_REGISTRY

export ONLY_DOCS
export PACKAGING_CHANGES
export GO_MOD_CHANGES

export KIND_VERSION
export KUBECONFIG

add_bin_path
