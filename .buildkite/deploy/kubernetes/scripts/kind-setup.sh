#!/usr/bin/env bash
set -euo pipefail

echo "--- Creating cluster"
kind create cluster --image kindest/node:${K8S_VERSION}
kubectl cluster-info
