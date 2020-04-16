#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
KIND_VERSION=${KIND_VERSION:?$MSG}
K8S_VERSION=${K8S_VERSION:?$MSG}
HOME=${HOME:?$MSG}
KIND_CMD="${HOME}/bin/kind"
KUBECTL_CMD="${HOME}/bin/kubectl"

mkdir -p "${HOME}/bin"

curl -sSLo "${KIND_CMD}" "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-$(uname -s)-$(uname -m)"
chmod +x "${KIND_CMD}"

curl -sSLo "${KUBECTL_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/$(uname -s)/$(uname -m)/kubectl"
chmod +x "${KUBECTL_CMD}"

