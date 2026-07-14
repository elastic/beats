#!/usr/bin/env bash

# Installs the kind version pinned via ASDF_KIND_VERSION using asdf.
#
# The metricbeat kubernetes module integration tests create a kind cluster
# (see dev-tools/mage/kubernetes/kind.go). Installing kind here means the tests
# rely on the version pinned in the pipeline env instead of whatever version
# happens to be baked into the CI VM image, so bumping the Kubernetes version
# does not require rebuilding and re-pinning the CI image.

set -euo pipefail

if [[ -z "${ASDF_KIND_VERSION:-}" ]]; then
  echo "ASDF_KIND_VERSION is not set; skipping kind installation"
  exit 0
fi

echo "--- Installing kind ${ASDF_KIND_VERSION} via asdf"
if ! asdf plugin list 2>/dev/null | grep -qx "kind"; then
  asdf plugin add kind
fi
asdf install kind "${ASDF_KIND_VERSION}"
asdf reshim kind

kind version
