#!/bin/bash

sudo usermod -a -G systemd-journal $USER

mkdir -p ./build/integration-tests/

sudo -u $USER bash <<EOF
groups | tee ./build/integration-tests/groups-from-script
whoami | tee ./build/integration-tests/whoami-from-script

source /opt/buildkite-agent/hooks/pre-command
source .buildkite/hooks/pre-command || echo "No pre-command hook found"
mage goIntegTest

EOF
