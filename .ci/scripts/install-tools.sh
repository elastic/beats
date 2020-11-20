#!/usr/bin/env bash
set -exuo pipefail

.ci/scripts/install-go.sh
.ci/scripts/install-docker-compose.sh
.ci/scripts/install-terraform.sh
(cd /tmp && go get gotest.tools/gotestsum)
make mage
