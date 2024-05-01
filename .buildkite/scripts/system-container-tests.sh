#!/bin/bash

set -euo pipefail

source .buildkite/scripts/common.sh

install_go_dependencies

# install docker

# DOCKER_VERSION="25.0"

# curl -fsSL https://get.docker.com -o get-docker.sh
# sudo sh ./get-docker.sh --version $DOCKER_VERSION

go test -timeout 20m -v ./tests