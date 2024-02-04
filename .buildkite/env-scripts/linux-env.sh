#!/usr/bin/env bash

source .buildkite/env-scripts/util.sh

DEBIAN_FRONTEND="noninteractive"

export DEBIAN_FRONTEND

sudo mkdir -p /etc/needrestart
echo "\$nrconf{restart} = 'a';" | sudo tee -a /etc/needrestart/needrestart.conf > /dev/null

# Remove this code once beats specific agent is set up
if [[ $PLATFORM_TYPE == "Linux" ]]; then
  echo ":: Installing libs ::"
  sudo apt-get update
  sudo apt-get install -y libsystemd-dev
  sudo apt install -y python3-pip
  sudo apt-get install -y python3-venv
fi

echo ":: Setting up environment ::"
add_bin_path
with_go
with_mage
