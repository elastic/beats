#!/bin/bash

set -euo pipefail

source .buildkite/scripts/common.sh

install_go_dependencies

gotestsum --format testname --junitfile junit-report.xml -- -v ./...
