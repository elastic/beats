#!/usr/bin/env bash

REPO="beats"
SETUP_GVM_VERSION="v0.5.1"
WORKSPACE="$(pwd)"
BIN="${WORKSPACE}/bin"
HW_TYPE="$(uname -m)"
PLATFORM_TYPE="$(uname)"
TMP_FOLDER="tmp.${REPO}"
ASDF_MAGE_VERSION="1.14.0"
SETUP_MAGE_VERSION="1.14.0"
DEBIAN_FRONTEND="noninteractive"

export SETUP_GVM_VERSION
export WORKSPACE
export BIN
export HW_TYPE
export PLATFORM_TYPE
export REPO
export TMP_FOLDER
export ASDF_MAGE_VERSION
export SETUP_MAGE_VERSION

if grep -q 'Ubuntu' /etc/*release; then
  export DEBIAN_FRONTEND
fi
