#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
DEFAULT_HOME="/usr/local"
K8S_VERSION=${K8S_VERSION:?$MSG}
HOME=${HOME:?$DEFAULT_HOME}
KUBECTL_CMD="${HOME}/bin/kubectl"

if command -v kubectl
then
    set +e
    echo "Found kubectl. Checking version.."
    FOUND_KUBECTL_VERSION=$(kubectl version --client --short 2>&1 >/dev/null | awk '{print $3}')
    if [ "${FOUND_KUBECTL_VERSION}" == "${K8S_VERSION}" ]
    then
        echo "Versions match. No need to install kubectl. Exiting."
        exit 0
    fi
    set -e
fi

echo "UNMET DEP: Installing kubectl"

mkdir -p "${HOME}/bin"

OS=$(uname -s| tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m| tr '[:upper:]' '[:lower:]')
if [ "${ARCH}" == "aarch64" ] ; then
    ARCH_SUFFIX=arm64
else
    ARCH_SUFFIX=amd64
fi

if curl -sSLo "${KUBECTL_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/${OS}/${ARCH_SUFFIX}/kubectl" ; then
    chmod +x "${KUBECTL_CMD}"
else
    echo "Something bad with the download, let's delete the corrupted binary"
    if [ -e "${KUBECTL_CMD}" ] ; then
        rm "${KUBECTL_CMD}"
    fi
    exit 1
fi

