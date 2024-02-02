#!/usr/bin/env bash

SETUP_GVM_VERSION="v0.5.1"
WORKSPACE="$(pwd)"
BIN="${WORKSPACE}/bin"
HW_TYPE="$(uname -m)"
PLATFORM_TYPE="$(uname)"

export SETUP_GVM_VERSION
export WORKSPACE
export BIN
export HW_TYPE
export PLATFORM_TYPE
