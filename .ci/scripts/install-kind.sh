#!/usr/bin/env bash
set -exuo pipefail

MSG="environment variable missing."
DEFAULT_HOME="/usr/local"
KIND_VERSION=${KIND_VERSION:?$MSG}
HOME=${HOME:?$DEFAULT_HOME}
KIND_CMD="${HOME}/bin/kind"

if command -v kind
then
    echo "Found Kind. Checking version.."
    FOUND_KIND_VERSION=$(kind --version 2>&1 >/dev/null | awk '{print $3}')
    if [ $FOUND_KIND_VERSION == $KIND_VERSION ]
    then
        echo "Versions match. No need to install Kind. Exiting."
        exit 0
    fi
fi

echo "UNMET DEP: Installing Kind"

mkdir -p "${HOME}/bin"

curl -sSLo "${KIND_CMD}" "https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-amd64"
chmod +x "${KIND_CMD}"
