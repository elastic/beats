#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
DEFAULT_HOME="/usr/local"
K8S_VERSION=${K8S_VERSION:?$MSG}
HOME=${HOME:?$DEFAULT_HOME}
KUBECTL_CMD="${HOME}/bin/kubectl"

if command -v kubectl
then
    echo "Found kubectl. Checking version.."
    FOUND_KUBECTL_VERSION=$(kubectl version --short 2>&1 >/dev/null | grep -i client | awk '{print $3}')
    if [ "${FOUND_KUBECTL_VERSION}" == "${K8S_VERSION}" ]
    then
        echo "Versions match. No need to install kubectl. Exiting."
        exit 0
    fi
fi

echo "UNMET DEP: Installing kubectl"

mkdir -p "${HOME}/bin"

if curl -sSLo "${KUBECTL_CMD}" "https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl" ; then
    chmod +x "${KUBECTL_CMD}"
else
    echo "Something bad with the download, let's delete the corrupted binary"
    if [ -e "${KUBECTL_CMD}" ] ; then
        rm "${KUBECTL_CMD}"
    fi
    exit 1
fi

