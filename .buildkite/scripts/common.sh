#!/bin/bash

set -euo pipefail

WORKSPACE="$(pwd)/bin"
REPO="beats"
platform_type=$(uname | tr '[:upper:]' '[:lower:]')
hw_type="$(uname -m)"

check_platform_architeture() {
# for downloading the terraform packages
  case "${hw_type}" in
   "x86_64")
        arch_type="amd64"
        ;;
    "aarch64")
        arch_type="arm64"
        ;;
    "arm64")
        arch_type="arm64"
        ;;
    *)
    echo "The current platform/OS type is unsupported yet"
    ;;
  esac
}

create_workspace() {
    if [[ ! -d "${WORKSPACE}" ]]; then
    mkdir -p ${WORKSPACE}
    fi
}

clean_workspace() {
    rm -rf ${WORKSPACE}
}

add_bin_path() {
    echo "Adding PATH to the environment variables..."
    create_workspace
    export PATH="${PATH}:${WORKSPACE}"
}

retry() {
    local retries=$1
    shift
    local count=0
    until "$@"; do
        exit=$?
        wait=$((2 ** count))
        count=$((count + 1))
        if [ $count -lt "$retries" ]; then
            >&2 echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
            sleep $wait
        else
            >&2 echo "Retry $count/$retries exited $exit, no more retries left."
            return $exit
        fi
    done
    return 0
}

with_Terraform() {
    echo "Setting up the Terraform environment..."
    local path_to_file="${WORKSPACE}/terraform.zip"
    create_workspace
    check_platform_architeture
    retry 5 curl -sSL -o ${path_to_file} "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_${platform_type}_${arch_type}.zip"
    unzip -q ${path_to_file} -d ${WORKSPACE}/
    rm ${path_to_file}
    chmod +x ${WORKSPACE}/terraform
    terraform version
}
