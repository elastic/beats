#!/usr/bin/env bash

set -euo pipefail

echo ":: Checking K8S ::"
cd deploy/kubernetes
make -C deploy/kubernetes all
make check-no-changes
