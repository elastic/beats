#!/bin/bash

set -euo pipefail

source .buildkite/scripts/common.sh

create_workspace

install_go_dependencies

mage notice
mage -v check
