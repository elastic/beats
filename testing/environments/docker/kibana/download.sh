#!/usr/bin/env bash
set -eo pipefail

if [ -z ${DOWNLOAD_URL+x} ]; then echo "DOWNLOAD_URL is unset"; exit 1; fi
if [ -z ${ELASTIC_VERSION+x} ]; then echo "ELASTIC_VERSION is unset"; exit 1; fi
if [ -z ${IMAGE_FLAVOR+x} ]; then echo "IMAGE_FLAVOR is unset"; exit 1; fi

url=${DOWNLOAD_URL}/kibana/kibana-oss/kibana-oss-${ELASTIC_VERSION}-linux-x86_64.tar.gz
if [ "${IMAGE_FLAVOR}" = "x-pack" ]; then
  url=${DOWNLOAD_URL}/kibana/kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz
fi

# Download.
curl -s -L -o kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz $url

# Validate SHA512.
expected_sha=$(curl -s -L $url.sha512 | awk '{print $1}')
observed_sha=$(sha512sum kibana-${ELASTIC_VERSION}-linux-x86_64.tar.gz | awk '{print $1}')
test "${expected_sha}" == "${observed_sha}"
