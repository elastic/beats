#!/bin/bash

#
# Wrapper script for starting the docker container.
#
# You must set AWS_ACCESS_KEY and AWS_SECRET_KEY in your environment prior to
# running. You can optionally pass the GPG key's passphrase as the environment
# variable PASS.
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

bucket="packages.elasticsearch.org"
prefix="beats"
dir="/beats-packer/build/upload"
gpg_key="/beats-packer/dev-tools/packer/docker/deb-rpm-s3/elasticsearch.asc"
origin=Elastic

docker run -it --rm \
  --env="PASS=$PASS" \
  --volume `pwd`/../../../..:/beats-packer \
  deb-rpm-s3 \
  --bucket=$bucket \
  --prefix=$prefix \
  --directory="$dir" \
  --aws-access-key="$AWS_ACCESS_KEY" \
  --aws-secret-key="$AWS_SECRET_KEY" \
  --gpg-key="$gpg_key" \
  --origin="$origin" \
  --verbose \
  "$@"

