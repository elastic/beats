#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

DEBIAN_FRONTEND="noninteractive"

set_env() {
  echo "--- Setting up environment"
  add_bin_path
  with_go
  with_mage
}

#if [[ $PLATFORM_TYPE == "Linux" ]]; then
#  check_platform_architecture
#  echo "ARCH: ${arch_type}"
#
#  # Remove this code once beats specific agent is set up
#  if grep -q 'Ubuntu' /etc/*release && [ "${arch_type}" == "arm64" ]; then
#    export DEBIAN_FRONTEND
#
#    sudo mkdir -p /etc/needrestart
#    echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf >/dev/null
#
#    echo "--- Ubuntu - Installing libs"
#    sudo apt-get update
#    sudo apt-get install -y libsystemd-dev
#    sudo apt install -y python3-pip
#    sudo apt-get install -y python3-venv
#
#    set_env
#  fi
#
#  # Remove this code once beats specific agent is set up
#  if grep -q 'Red Hat' /etc/*release; then
#    echo "--- RHL - Installing libs"
#    sudo yum update -y
#    sudo yum install -y systemd-devel
#    sudo yum install -y python3-pip
#    sudo yum install -y python3
#    pip3 install virtualenv
#
#    set_env
#  fi
#fi

if [[ $PLATFORM_TYPE == Darwin* ]]; then
  if [[ "$BUILDKITE_STEP_KEY" == macos* ]]; then
    ulimit -Sn 30000
  fi

  set_env
fi
