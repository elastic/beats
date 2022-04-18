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

echo 'List all the files'
find build/distributions -type f -ls || true
