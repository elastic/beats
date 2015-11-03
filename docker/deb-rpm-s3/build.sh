#!/bin/bash
set -e

#
# Build script for the deb-rpm-s3 docker container.
#

cd "$(dirname "$0")"

docker build -t deb-rpm-s3 .
