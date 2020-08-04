#!/usr/bin/env bash

set -exuo pipefail

MSG="environment variable missing."
TERRAFORM_VERSION=${TERRAFORM_VERSION:?$MSG}
HOME=${HOME:?$MSG}
TERRAFORM_CMD="${HOME}/bin/terraform"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if command -v terraform
then
    echo "Found Terraform. Checking version.."
    FOUND_TERRAFORM_VERSION=$(terraform --version | awk '{print $2}' | sed s/v//)
    if [ $FOUND_TERRAFORM_VERSION == $TERRAFORM_VERSION ]
    then
        echo "Versions match. No need to install Terraform. Exiting."
        exit 0
    fi
fi

echo "UNMET DEP: Installing Terraform"

mkdir -p "${HOME}/bin"

curl -sSLo - "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${OS}_amd64.zip" > ${TERRAFORM_CMD}.zip
unzip -o ${TERRAFORM_CMD}.zip -d $(dirname ${TERRAFORM_CMD})
rm ${TERRAFORM_CMD}.zip

chmod +x "${TERRAFORM_CMD}"
