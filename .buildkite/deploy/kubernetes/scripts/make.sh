#!/usr/bin/env bash

set -euo pipefail

#source .buildkite/env-scripts/unix-env.sh

echo ":: Checking K8S ::"
cd deploy/kubernetes
make -C deploy/kubernetes all
make check-no-changes
