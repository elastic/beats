#!/usr/bin/env bash

set -exuo pipefail

MSG="environment variable missing."
TERRAFORM_VERSION=${TERRAFORM_VERSION:?$MSG}
HOME=${HOME:?$MSG}
TERRAFORM_CMD="${HOME}/bin/terraform"

if command -v terraform
then
    set +e
    echo "Found Terraform. Checking version.."
    FOUND_TERRAFORM_VERSION=$(terraform --version | awk '{print $2}' | sed s/v//)
    if [ "$FOUND_TERRAFORM_VERSION" == "$TERRAFORM_VERSION" ]
    then
        echo "Versions match. No need to install Terraform. Exiting."
        exit 0
    fi
    set -e
fi

echo "UNMET DEP: Installing Terraform"

mkdir -p "${HOME}/bin"

OS=$(uname -s| tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m| tr '[:upper:]' '[:lower:]')
if [ "${ARCH}" == "aarch64" ] ; then
    ARCH_SUFFIX=arm64
elif [ "${ARCH}" == "x86_64" ] ; then
    ARCH_SUFFIX=amd64
elif [ "${ARCH}" == "i686" ] ; then
    ARCH_SUFFIX=386
else
    ARCH_SUFFIX=arm
fi

if curl -sSLo - "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${OS}_${ARCH_SUFFIX}.zip" > "${TERRAFORM_CMD}.zip" ; then
    unzip -o "${TERRAFORM_CMD}".zip -d "$(dirname ${TERRAFORM_CMD})"
    rm "${TERRAFORM_CMD}".zip

    chmod +x "${TERRAFORM_CMD}"
else
    echo "Something bad with the download, let's delete the corrupted binary"
    if [ -e "${TERRAFORM_CMD}" ] ; then
        rm "${TERRAFORM_CMD}"
    fi
    exit 1
fi
