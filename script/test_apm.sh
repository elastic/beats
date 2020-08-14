#!/usr/bin/env bash

# This script allows to test against the code from apm-server
# By default it is checked against master. The env variable BRANCH
# can be set to any value accepted by git checkout.
#
# go build is executed on the apm-server code to verify that it can
# still be built and runs the go tests with make unit.

set -e

# Check if a special branch env variable is set
if [ -z "${BRANCH}" ]; then
    BRANCH=master
fi

echo "apm-server branch: $BRANCH"

BASE_PATH=$(pwd)
BUILD_DIR=$BASE_PATH/build
ELASTIC_DIR=$BUILD_DIR/apm-test/src/github.com/elastic
APM_SERVER_DIR=$ELASTIC_DIR/apm-server

# Cleanup and create directories
rm -rf $APM_SERVER_DIR
mkdir -p $APM_SERVER_DIR

# Clone and checkout defined branch
git clone https://github.com/elastic/apm-server $APM_SERVER_DIR
cd $APM_SERVER_DIR
git checkout $BRANCH
cd $BASE_PATH

# Replace libbeat with local libbeat version
rm -r $APM_SERVER_DIR/vendor/github.com/elastic/beats/libbeat
cp -r libbeat $APM_SERVER_DIR/vendor/github.com/elastic/beats/

cd $APM_SERVER_DIR

echo "Build apm-server binary"

# Set temporary GOPATH to make sure local version of libbeat is used
GOPATH=$BUILD_DIR/apm-test go build

echo "Run apm-server unit tests"
GOPATH=$BUILD_DIR/apm-test make unit
