#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
DEFAULT_HOME="/usr/local"
K8S_VERSION=${K8S_VERSION:?$MSG}
HOME=${HOME:?$DEFAULT_HOME}
KUBECTL_CMD="${HOME}/bin/kubectl"

mkdir -p "${HOME}/bin"

curl -sSLo "${KUBECTL_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/$(uname -s)/$(uname -m)/kubectl"
chmod +x "${KUBECTL_CMD}"

