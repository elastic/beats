#!/usr/bin/env bash

set -exuo pipefail

MSG="parameter missing."
TERRAFORM_VERSION=${TERRAFORM_VERSION:?$MSG}
HOME=${HOME:?$MSG}
TERRAFORM_CMD="${HOME}/bin/terraform"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

mkdir -p "${HOME}/bin"

curl -sSLo - "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${OS}_amd64.zip" > ${TERRAFORM_CMD}.zip
unzip -o ${TERRAFORM_CMD}.zip -d $(dirname ${TERRAFORM_CMD})
rm ${TERRAFORM_CMD}.zip

chmod +x "${TERRAFORM_CMD}"
