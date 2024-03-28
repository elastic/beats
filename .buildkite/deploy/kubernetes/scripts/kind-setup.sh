#!/usr/bin/env bash
set -euo pipefail

kind create cluster --image kindest/node:${K8S_VERSION}

echo "Cluster info: "
kubectl cluster-info
