#!/usr/bin/env bash
set -exuo pipefail

echo "going to setup kind:"
echo $K8S_VERSION
kind create cluster --image kindest/node:${K8S_VERSION}
kubectl cluster-info
