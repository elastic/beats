#!/usr/bin/env bash
set -exuo pipefail

kind create cluster --image kindest/node:${K8S_VERSION}
kubectl cluster-info
