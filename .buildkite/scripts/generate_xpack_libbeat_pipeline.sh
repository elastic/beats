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
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Ubuntu Unit Tests"

      - label: ":go: Go Integration Tests"
        key: "mandatory-int-test"
        command: "cd $BEATS_PROJECT_NAME && mage goIntegTest"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Go Integration Tests"

      - label: ":python: Python Integration Tests"
        key: "mandatory-python-int-test"
        command: "cd $BEATS_PROJECT_NAME && mage pythonIntegTest"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Python Integration Tests"

      - label: ":windows: Windows 2016 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage -w reader\etw build goUnitTest
        key: "mandatory-win-2016-unit-tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2016 Unit Tests"

      - label: ":windows: Windows 2022 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage -w reader\etw build goUnitTest
        key: "mandatory-win-2022-unit-tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2022 Unit Tests"

### TODO: this condition will be changed in the Phase 3 of the Migration Plan https://docs.google.com/document/d/1IPNprVtcnHlem-uyGZM0zGzhfUuFAh4LeSl9JFHMSZQ/edit#heading=h.sltz78yy249h
  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:
      - label: ":windows: Windows 10 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage -w reader\etw build goUnitTest
        key: "extended-win-10-unit-tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 10 Unit Tests"

      - label: ":windows: Windows 11 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage -w reader\etw build goUnitTest
        key: "extended-win-11-unit-tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 11 Unit Tests"

      - label: ":windows: Windows 2019 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage -w reader\etw build goUnitTest
        key: "extended-win-2019-unit-tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2019 Unit Tests"

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
      - label: ":linux: Ubuntu ARM64 Unit Tests"
        key: "extended-arm64-unit-tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Ubuntu ARM64 Unit Tests"

YAML
fi

echo "+++ Printing dynamic steps"
cat $pipelineName | yq . -P

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
