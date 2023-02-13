#!/usr/bin/env bash
#
# this file is run daily to generate worker packer images
#

# shellcheck disable=SC1091
source /usr/local/bin/bash_standard_lib.sh

# shellcheck disable=SC1091
source ./dev-tools/common.bash

######################
############ FUNCTIONS
######################
function getBeatsVersion() {
  grep 'defaultBeatVersion' libbeat/version/version.go | cut -d= -f2 | sed 's#"##g' | tr -d " "
}

function dockerPullCommonImages() {
  DOCKER_IMAGES="docker.elastic.co/observability-ci/database-instantclient:12.2.0.1
  docker.elastic.co/observability-ci/database-enterprise:12.2.0.1
  docker.elastic.co/beats-dev/fpm:1.11.0
  golang:1.14.12-stretch
  ubuntu:20.04
  "
  for image in ${DOCKER_IMAGES} ; do
    (retry 2 docker pull ${image}) || echo "Error pulling ${image} Docker image. Continuing."
  done
  docker tag \
    docker.elastic.co/observability-ci/database-instantclient:12.2.0.1 \
    store/oracle/database-instantclient:12.2.0.1 \
    || echo "Error setting the Oracle Instant Client tag"
  docker tag \
    docker.elastic.co/observability-ci/database-enterprise:12.2.0.1 \
    store/oracle/database-enterprise:12.2.0.1 \
    || echo "Error setting the Oracle Database tag"
}

function dockerPullImages() {
  SNAPSHOT=$1
  get_go_version

  DOCKER_IMAGES="
  docker.elastic.co/elasticsearch/elasticsearch:${SNAPSHOT}
  docker.elastic.co/kibana/kibana:${SNAPSHOT}
  docker.elastic.co/logstash/logstash:${SNAPSHOT}
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-arm
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-armhf
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-armel
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-base-arm-debian9
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-darwin
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main-debian7
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main-debian8
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-main-debian9
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-mips
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-ppc
  docker.elastic.co/beats-dev/golang-crossbuild:${GO_VERSION}-s390x
  golang:${GO_VERSION}
  docker.elastic.co/infra/release-manager:latest
  "
  for image in ${DOCKER_IMAGES}
  do
    (retry 2 docker pull "${image}") || echo "Error pulling ${image} Docker image. Continuing."
  done
}

#################
############ MAIN
#################
if [ -x "$(command -v docker)" ]; then
  set -x
  echo "Docker pull common docker images"
  dockerPullCommonImages

  ## GitHub api returns up to 100 entries.
  ## Probably we need a different approach to search the latest minor.
  latest7Minor=$(curl -s https://api.github.com/repos/elastic/beats/branches\?per_page=100 | jq -r '.[].name' | grep "^7." | tail -1)
  latest8Minor=$(curl -s https://api.github.com/repos/elastic/beats/branches\?per_page=100 | jq -r '.[].name' | grep "^8." | tail -1)

  for branch in main $latest7Minor $latest8Minor; do
    if [ "$branch" != "main" ] ; then
      echo "Checkout the branch $branch"
      git checkout "$branch"
    fi

    VERSION=$(getBeatsVersion)
    dockerPullImages "${VERSION}-SNAPSHOT"
  done
fi
