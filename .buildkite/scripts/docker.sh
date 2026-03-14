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

# Print docker login information for debugging purposes
# See https://docs.docker.com/docker-hub/usage/pulls/#view-pull-rate-and-limit
docker_info() {
  echo "~~~ Docker Login Status and Rate Limits"

  # Check if we're logged in to Docker Hub
  # see https://stackoverflow.com/a/47580834
  if docker login; then
    echo "✅ Logged in to Docker Hub"
  else
    echo "⚠ Not logged in to Docker Hub"
  fi

  # Get authentication token (works for both authenticated and anonymous)
  TOKEN=$(curl -s "https://auth.docker.io/token?service=registry.docker.io&scope=repository:ratelimitpreview/test:pull" | jq -r .token 2>/dev/null || echo "")

  if [ -n "$TOKEN" ]; then
    echo "Fetching rate limit information from Docker Hub"
    curl -s --head -H "Authorization: Bearer $TOKEN" https://registry-1.docker.io/v2/ratelimitpreview/test/manifests/latest | grep -i "rate" || true
  else
    echo "⚠ Unable to authenticate with Docker Hub to check rate limits"
  fi
}

docker_disable_containerd_snapshotter
docker_info
