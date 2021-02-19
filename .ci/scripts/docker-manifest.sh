#!/usr/bin/env bash
set -exuo pipefail

MSG="parameter missing."
TEMPLATE_IMAGE=${1:?$MSG}
IMAGE=${2:?$MSG}
PLATFORMS=${3:-"linux/amd64,linux/arm64"}
HOME=${HOME:?$MSG}

##############################################
###### INSTALL manifest tool
##############################################
OS=$(uname -s| tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m| tr '[:upper:]' '[:lower:]')
MT_CMD="${HOME}/bin/manifest-tool"

if [ "${ARCH}" == "aarch64" ] ; then
    MT_ARCH_SUFFIX=arm64
elif [ "${ARCH}" == "x86_64" ] ; then
    MT_ARCH_SUFFIX=amd64
elif [ "${ARCH}" == "i686" ] ; then
    MT_ARCH_SUFFIX=386
else
    MT_ARCH_SUFFIX=armv7
fi

curl -sSLo "${MT_CMD}" "https://github.com/estesp/manifest-tool//releases/download/v1.0.3/manifest-tool-${OS}-${MT_ARCH_SUFFIX}"
chmod +x "${MT_CMD}"

##############################################
###### CREATE manifest
##############################################
"${MT_CMD}" \
    push \
    from-args \
    --platforms "${PLATFORMS}" \
    --template "${TEMPLATE_IMAGE}" \
    --target "${IMAGE}"
