#!/bin/bash
junitfile=$1 # filename for jnit annotation plugin

set -euo pipefail

source .buildkite/scripts/common.sh

install_go_dependencies

gotestsum --format testname --junitfile "${junitfile:-junit-lin-report.xml}" -- -v ./...
