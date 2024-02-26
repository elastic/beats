#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.libbeat-dynamic.yml"

echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML

steps:

  - group: "Mandatory Tests"
    key: "mandatory-tests"
    steps:
      - label: ":linux: Ubuntu Unit Tests"
        key: "mandatory-linux-unit-test"
        command: ".buildkite/scripts/unit_tests.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

      - label: ":go: Go Integration Tests"
        key: "mandatory-int-test"
        command: ".buildkite/scripts/go_int_tests.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

      - label: ":python: Python Integration Tests"
        key: "mandatory-python-int-test"
        command: ".buildkite/scripts/py_int_tests.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

      - label: ":windows: Windows Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1 unittests"
        key: "mandatory-win-unit-tests"
        agents:
          provider: "gcp"
          image: "{{matrix.image}}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_2016}"
              - "${IMAGE_WIN_2022}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

### TODO: this condition will be changed in the Phase 3 of the Migration Plan https://docs.google.com/document/d/1IPNprVtcnHlem-uyGZM0zGzhfUuFAh4LeSl9JFHMSZQ/edit#heading=h.sltz78yy249h
  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:
      - label: ":windows: Win 2019 Unit Tests"
        key: "extended-win-2019-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1 unittests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows: Windows 10 Unit Tests"
        key: "extended-win-10-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1 unittests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows: Windows 11 Unit Tests"
        key: "extended-win-11-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1 unittests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

echo "Check and add the Extended Tests into the pipeline"
if are_conditions_met_arm_tests; then
  cat >> $pipelineName <<- YAML

  - group: "Extended Tests"
    key: "extended-tests"
    steps:
      - label: ":linux: Arm64 Unit Tests"
        key: "extended-arm64-unit-tests"
        command: ".buildkite/scripts/unit_tests.sh"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
