#!/usr/bin/env bash
set -exuo pipefail

.ci/scripts/install-docker-compose.sh
.ci/scripts/install-terraform.sh
