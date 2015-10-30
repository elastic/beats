#!/bin/bash

#
# Wrapper script for starting the docker container.
#
# You must set AWS_ACCESS_KEY and AWS_SECRET_KEY in your environment prior to
# running. You can optionally pass the GPG key's passphrase as the environment
# variable PASS.
#

cd "$(dirname "$0")"

docker run -it --rm \
  --env="PASS=$PASS" \
  --volume `pwd`/../..:/beats-packer \
  deb-rpm-s3 \
  --bucket=packages.elasticsearch.org \
  --prefix=beats \
  --directory=/beats-packer/build/upload \
  --aws-access-key="$AWS_ACCESS_KEY" \
  --aws-secret-key="$AWS_SECRET_KEY" \
  --verbose \
  "$@"

