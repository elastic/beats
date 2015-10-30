#!/bin/bash
set -e

#
# Build script for the deb-rpm-s3 docker container.
#

cd "$(dirname "$0")"

if [ ! -e "elasticsearch.asc" ]; then
    cat << EOF
You must place a copy of the Elasticsearch GPG signing key (named
elasticsearch.asc) into
  
  $PWD

prior to building this docker image.

EOF
    exit 1
fi

docker build -t deb-rpm-s3 .
