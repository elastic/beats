#!/bin/bash

set -euo pipefail

source .buildkite/scripts/common.sh

create_workspace

#with_go

install_go_dependencies

mage -v check
