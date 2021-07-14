#!/usr/bin/env bash
set +e
readonly LOCATION="${1?Please define the path where the fix permissions should run from}"

if ! docker version 2>&1 >/dev/null ; then
  echo "It requires Docker daemon to be installed and running"
else
  ## Detect architecture to support ARM specific docker images.
  ARCH=$(uname -m| tr '[:upper:]' '[:lower:]')
  if [ "${ARCH}" == "aarch64" ] ; then
    DOCKER_IMAGE=arm64v8/alpine:3
  else
    DOCKER_IMAGE=alpine:3.4
  fi
  set -e
  echo "Change ownership of all files inside the specific folder from root/root to current user/group"
  set -x
  docker run -v "${LOCATION}":/beat ${DOCKER_IMAGE} sh -c "find /beat -user 0 -exec chown -h $(id -u):$(id -g) {} \;"
fi

set -e
echo "Change permissions with write access of all files inside the specific folder"
chmod -R +w "${LOCATION}"
