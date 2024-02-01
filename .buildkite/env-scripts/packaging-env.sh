#!/usr/bin/env bash

source .buildkite/env-scripts/util.sh

docs_changeset="^.*\.(asciidoc|md)$
deploy/kubernetes/.*-kubernetes.yaml"
packaging_changeset="^dev-tools/packaging/
^.go-version"

# Env vars for Packaging stage
DOCKER_REGISTRY="docker.elastic.co"
SNAPSHOT=true
VERSION=$(make get-version | tr -d '\n')
ONLY_DOCS=$(changeset_applies "$docs_changeset")
PACKAGING_CHANGES=$(changeset_applies "$packaging_changeset")
GO_MOD_CHANGES=$(changeset_applies "^go.mod")

export PACKAGING_CHANGES
export ONLY_DOCS
export GO_MOD_CHANGES
export DOCKER_REGISTRY
export SNAPSHOT
export VERSION
export REPO
