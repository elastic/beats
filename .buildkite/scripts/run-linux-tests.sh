#!/bin/bash

set -euox pipefail

source .buildkite/scripts/common.sh

#create_workspace

#with_go

which go
echo "$(go env GOPATH)"
echo "$PATH"
pwd

install_go_dependencies

go clean -modcache

gotestsum --format testname --junitfile 'junit-report.xml' -- '-v ./...'
