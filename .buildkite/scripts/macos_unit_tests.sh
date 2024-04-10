#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

if [[ -z "${GO_VERSION-""}" ]]; then
  GO_VERSION=$(cat "${WORKSPACE}/.go-version")
  export GO_VERSION
fi

add_bin_path
with_go "${GO_VERSION}"
with_mage
with_python
with_dependencies
config_git
with_macos_docker_compose

if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-heartbeat" ]]; then
  # Install NodeJS
  withNodeJSEnv "${NODEJS_VERSION}"
  installNodeJsDependencies

  echo "Install @elastic/synthetics"
  npm i -g @elastic/synthetics
fi

mage dumpVariables

#sudo command doesn't work at the "pre-command" hook because of another user environment (root with strange permissions)
sudo chmod -R go-w "${BEATS_PROJECT_NAME}/"     #TODO: Remove when the issue is solved https://github.com/elastic/beats/issues/37838

pushd "${BEATS_PROJECT_NAME}" > /dev/null

#TODO "umask 0022" has to be removed after our own image is ready (it has to be moved to the image)
umask 0022    # fix the filesystem permissions issue like this: https://buildkite.com/elastic/beats-metricbeat/builds/1329#018d3179-25a9-475b-a2c8-64329dfe092b/320-1696

echo "--- Run Unit Tests"
mage build unitTest

popd > /dev/null
