#!/usr/bin/env bash
set -exuo pipefail

.ci/scripts/install-go.sh
.ci/scripts/install-docker-compose.sh
.ci/scripts/install-terraform.sh
go get gotest.tools/gotestsum
go mod tidy
make mage
