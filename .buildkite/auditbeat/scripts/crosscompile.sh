#!/usr/bin/env bash

set -euo pipefail

source .buildkite/env-scripts/unix-env.sh

echo ":: Executing Crosscompile ::"
make -C auditbeat crosscompile
