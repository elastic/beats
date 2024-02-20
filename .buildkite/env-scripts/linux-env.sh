#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/util.sh

DEBIAN_FRONTEND="noninteractive"

sudo mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null

if [[ $PLATFORM_TYPE == "Linux" ]]; then
  # Remove this code once beats specific agent is set up
  if grep -q 'Ubuntu' /etc/*release; then
    export DEBIAN_FRONTEND

    echo "--- Ubuntu - Installing libs"
    sudo apt-get update
    sudo apt-get install -y libsystemd-dev
    sudo apt install -y python3-pip
    sudo apt-get install -y python3-venv
  fi

  # Remove this code once beats specific agent is set up
  if grep -q 'Red Hat' /etc/*release; then
    echo "--- RHL - Installing libs"
    sudo yum update -y
    sudo yum install -y systemd-devel
    sudo yum install -y python3-pip
    sudo yum install -y python3
    pip3 install virtualenv
  fi
fi

if [[ $PLATFORM_TYPE == Darwin* ]]; then
  echo "--- Setting larger ulimit on MacOS"
  # To bypass file descriptor errors like "Too many open files error" on MacOS
  ulimit -Sn 50000
  echo "--- ULIMIT: $(ulimit -n)"
fi

echo "--- Setting up environment"
add_bin_path
with_go
with_mage
