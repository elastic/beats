#!/usr/bin/env bash
set -euo pipefail
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends --yes msitools