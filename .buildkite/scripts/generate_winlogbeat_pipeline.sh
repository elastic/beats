#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.winlogbeat-dynamic.yml"

echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML

steps:

  - group: "Mandatory Tests"
    key: "mandatory-tests"
    steps:

      - label: ":negative_squared_cross_mark: Cross compile"
        key: "mandatory-cross-compile"
        command: ".buildkite/scripts/crosscompile.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows: Windows 2016/2019/2022 Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        key: "mandatory-win-unit-tests"
        agents:
          provider: "gcp"
          image: "{{matrix.image}}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_2016}"
              - "${IMAGE_WIN_2019}"
              - "${IMAGE_WIN_2022}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

# echo "Add the extended windows tests into the pipeline"
# TODO: ADD conditions from the main pipeline

  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:

      - label: ":windows: Windows 10 Unit Tests"
        key: "extended-win-10-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows: Windows 11 Unit Tests"
        key: "extended-win-11-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

echo "Check and add the Packaging into the pipeline"
if are_conditions_met_packaging; then
  cat >> $pipelineName <<- YAML

  - wait: ~
    depends_on:
      - step: "mandatory-tests"
        allow_failure: false

  - group: "Packaging"    # TODO: check conditions for future the main pipeline migration: https://github.com/elastic/beats/pull/28589
    key: "packaging"
    steps:
      - label: ":linux: Packaging Linux"
        key: "packaging-linux"
        command: ".buildkite/scripts/packaging.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        env:
          PLATFORMS: "+all linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64"


YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
