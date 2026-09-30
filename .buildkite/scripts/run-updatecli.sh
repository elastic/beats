#!/usr/bin/env bash
# Installs updatecli (version pinned in .updatecli-version) and applies the
# given updatecli pipeline config.
#
# Usage: run-updatecli.sh <path-to-config>
#
# Required environment:
#   GITHUB_TOKEN (exported by the elastic/vault-github-token Buildkite plugin)
set -euo pipefail

CONFIG=${1:?path to an updatecli config file is required}

UPDATECLI_VERSION=$(cat "${WORKSPACE}/.updatecli-version")

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          UPDATECLI_ARCH="x86_64" ;;
  aarch64 | arm64) UPDATECLI_ARCH="arm64" ;;
  *) echo "Unsupported architecture: ${ARCH}" >&2; exit 1 ;;
esac

BIN=$(mktemp -d)
trap 'rm -rf "${BIN}"' EXIT

echo "--- Installing updatecli ${UPDATECLI_VERSION}"
curl -fsSL -o "${BIN}/updatecli.tar.gz" \
  "https://github.com/updatecli/updatecli/releases/download/${UPDATECLI_VERSION}/updatecli_Linux_${UPDATECLI_ARCH}.tar.gz"
tar -xzf "${BIN}/updatecli.tar.gz" -C "${BIN}" updatecli
chmod +x "${BIN}/updatecli"

echo "--- Running updatecli apply against ${CONFIG}"
"${BIN}/updatecli" apply --config "${CONFIG}"
