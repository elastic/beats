#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
DEFAULT_HOME="/usr/local"
KIND_VERSION=${KIND_VERSION:?$MSG}
HOME=${HOME:?$DEFAULT_HOME}
KIND_CMD="${HOME}/bin/kind"

mkdir -p "${HOME}/bin"

curl -sSLo "${KIND_CMD}" "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-amd64"
chmod +x "${KIND_CMD}"
