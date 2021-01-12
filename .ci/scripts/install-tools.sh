#!/usr/bin/env bash
set -exuo pipefail

.ci/scripts/install-go.sh
.ci/scripts/install-docker-compose.sh
.ci/scripts/install-terraform.sh
make mage
