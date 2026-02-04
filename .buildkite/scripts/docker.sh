#!/usr/bin/env bash
#
# It disables docker containerd snapshotter in CI
#
# For further details, see https://github.com/elastic/elastic-agent/issues/11604
#
set -euo pipefail

# Disable the containerd snapshotter, as it affects the output of docker save.
# See https://github.com/elastic/elastic-agent/issues/11604
docker_disable_containerd_snapshotter() {
  if ! systemctl is-enabled docker; then
    return 0
  fi
  cat << EOF | sudo tee /etc/docker/daemon.json >/dev/null
{
  "features": {
    "containerd-snapshotter": false
  }
}
EOF
  sudo systemctl restart docker
}

docker_disable_containerd_snapshotter