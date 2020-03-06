#!/bin/bash

dir_resolve()
{
    cd "$1" 2>/dev/null || return $?  # cd to desired directory; if fail, quell any error messages but return exit status
    echo "`pwd -P`" # output full, link-resolved path
}

set -e

TARGET=`dirname $0`
TARGET=`dir_resolve $TARGET`
cd $TARGET

go get github.com/golang/protobuf/{proto,protoc-gen-go}


tmp_dir=$(mktemp -d)
mkdir -p $tmp_dir/loggregator

cp $GOPATH/src/github.com/cloudfoundry/loggregator-api/v2/*proto $tmp_dir/loggregator

protoc $tmp_dir/loggregator/*.proto --go_out=plugins=grpc:. --proto_path=$tmp_dir/loggregator

rm -r $tmp_dir
