#!/usr/bin/env bash

set -euo pipefail

echo "--- Executing Crosscompile"
make -C auditbeat crosscompile
