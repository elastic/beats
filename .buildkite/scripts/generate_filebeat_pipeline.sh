#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.filebeat-dynamic.yml"

# TODO: steps: must be always included
echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML


steps:
  - group: "Filebeat Mandatory Testing"
    key: "mandatory-tests"
    if: build.env("GITHUB_PR_TRIGGER_COMMENT") == "filebeat" || build.env("BUILDKITE_PULL_REQUEST") != "false"

    steps:
      - label: ":ubuntu: Unit Tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Filebeat: linux/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "filebeat/build/*.xml"
          - "filebeat/build/*.json"

      - label: ":ubuntu: Go Integration Tests"
        command: "cd $BEATS_PROJECT_NAME && mage goIntegTest"
        notify:
          - github_commit_status:
              context: "Filebeat: Go Integration Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "filebeat/build/*.xml"
          - "filebeat/build/*.json"

      - label: ":ubuntu: Python Integration Tests"
        command: "cd $BEATS_PROJECT_NAME && mage pythonIntegTest"
        notify:
          - github_commit_status:
              context: "Filebeat: Python Integration Tests"
        agents:
          provider: gcp
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "filebeat/build/*.xml"
          - "filebeat/build/*.json"

      - label: ":windows: Windows 2016/2022 Unit Tests - {{matrix.image}}"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
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
              - "${IMAGE_WIN_2022}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

echo "Check and add the Extended Tests into the pipeline"

if are_conditions_met_arm_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Extended Tests: ARM"
      key: "extended-tests-arm"
      steps:
      - label: ":linux: ARM64 Unit Tests"
        key: "arm-extended"
        command:
          - ".buildkite/filebeat/scripts/unit-tests.sh"
        notify:
          - github_commit_status:
              context: "Filebeat/Extended: Unit Tests ARM"
        agents:
          provider: "aws"
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths: "filebeat/build/*.xml"

YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:
      - label: ":windows: Windows Unit Tests - {{matrix.image}}"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        key: "extended-win-unit-tests"
        agents:
          provider: "gcp"
          image: "{{matrix.image}}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_10}"
              - "${IMAGE_WIN_11}"
              - "${IMAGE_WIN_2019}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
YAML
fi

echo "Check and add the Packaging into the pipeline"
if are_conditions_met_packaging; then
cat >> $pipelineName <<- YAML
  - group: "Packaging"    # TODO: check conditions for future the main pipeline migration: https://github.com/elastic/beats/pull/28589
    key: "packaging"
    depends_on:
          - "mandatory-tests"
    steps:
      - label: ":linux: Packaging Linux"
        key: "packaging-linux"
        command: "cd $BEATS_PROJECT_NAME && mage package"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        env:
          PLATFORMS: "${PACKAGING_PLATFORMS}"

      - label: ":linux: Packaging ARM"
        key: "packaging-arm"
        command: "cd $BEATS_PROJECT_NAME && mage package"
        agents:
          provider: "aws"
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        env:
          PLATFORMS: "${PACKAGING_ARM_PLATFORMS}"
          PACKAGES: "docker"
YAML
fi

echo "+++ Printing dynamic steps"
cat $pipelineName | yq . -P

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
