#!/usr/bin/env bash
#
# This script is executed by the DRA stage.
# It prepares the required files to be consumed by the release-manager
# It can be published as snapshot or staging, for such you use
# the paramater $0 "snapshot" or $0 "staging"
#
set -ueo pipefail

readonly TYPE=${1:-snapshot}

# rename dependencies.csv to the name expected by release-manager.
VERSION=$(make get-version)
FINAL_VERSION=$VERSION-SNAPSHOT
if [ "$TYPE" != "snapshot" ] ; then
  FINAL_VERSION=$VERSION
fi
echo "Rename dependencies to $FINAL_VERSION"
mv build/distributions/dependencies.csv \
   build/distributions/dependencies-"$FINAL_VERSION".csv

# rename docker files to support the unified release format.
# TODO: this could be supported by the package system itself
#       or the unified release process the one to do the transformation
#       See https://github.com/elastic/beats/pull/30895
find build/distributions -name '*linux-arm64.docker.tar.gz*' -print0 |
  while IFS= read -r -d '' file
  do
    echo "Rename file $file"
    mv "$file" "${file/linux-arm64.docker.tar.gz/docker-image-linux-arm64.tar.gz}"
  done

find build/distributions -name '*linux-amd64.docker.tar.gz*' -print0 |
  while IFS= read -r -d '' file
  do
    echo "Rename file $file"
    mv "$file" "${file/linux-amd64.docker.tar.gz/docker-image-linux-amd64.tar.gz}"
  done

echo 'List all the files'
find build/distributions -type f -ls || true
