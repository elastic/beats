#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
K8S_VERSION=${K8S_VERSION:?$MSG}
HOME=${HOME:?$MSG}
KBC_CMD="${HOME}/bin/kubectl"

mkdir -p "${HOME}/bin"

curl -sSLo "${KBC_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl"
chmod +x "${KBC_CMD}"

GO111MODULE="on" go get sigs.k8s.io/kind@v0.5.1
kind create cluster --image kindest/node:${K8S_VERSION}

export KUBECONFIG="$(kind get kubeconfig-path)"
kubectl cluster-info
