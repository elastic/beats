#!/usr/bin/env bash
set -exuo pipefail

kind create cluster --image kindest/node:${K8S_VERSION} --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    scheduler:
      extraArgs:
        bind-address: "0.0.0.0"
        port: "10251"
        secure-port: "10259"
    controllerManager:
      extraArgs:
        bind-address: "0.0.0.0"
        port: "10252"
        secure-port: "10257"
EOF
kubectl cluster-info
