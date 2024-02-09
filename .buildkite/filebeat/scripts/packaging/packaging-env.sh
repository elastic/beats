#!/usr/bin/env bash

source .buildkite/env-scripts/util.sh

docs_changeset="^.*\.(asciidoc|md)$
deploy/kubernetes/.*-kubernetes.yaml"
packaging_changeset="^dev-tools/packaging/
^.go-version"

VERSION=$(make get-version | tr -d '\n')
ONLY_DOCS=$(changeset_applies "$docs_changeset")
PACKAGING_CHANGES=$(changeset_applies "$packaging_changeset")
GO_MOD_CHANGES=$(changeset_applies "^go.mod")
# Change the postfix to -SNAPSHOT, once Jenkins is disabled


export PACKAGING_CHANGES
export ONLY_DOCS
export GO_MOD_CHANGES
export DOCKER_REGISTRY
export VERSION
export REPO
#export IMG_POSTFIX


#VARIANTS -> foreach = IMAGES: image map with SOURCE, TARGET, ARCH

#IMAGES -> foreach = tag and push
# registry: ${REGISTRY},
# sourceTag: calculate_tags->sourceTag,
# targetTag: "${tag}" (non arm) // ${tag}-${image.arch} (arm) --> foreach $TAGS
# source: ${SOURCE},
# target: ${TARGET}
