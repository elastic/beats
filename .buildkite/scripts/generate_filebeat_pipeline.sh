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

      - label: ":windows:-2016 Unit Tests"
        key: "windows-2016-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Filebeat: windows 2016/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows:-2022 Unit Tests"
        key: "windows-2022-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Filebeat: windows 2022/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

echo "Check and add the Extended Tests into the pipeline"

if are_conditions_met_arm_tests || are_conditions_met_macos_tests; then
  cat >> $pipelineName <<- YAML

  - group: "Extended Tests"
    key: "extended-tests"
    steps:
YAML
fi

if are_conditions_met_macos_tests; then
  cat >> $pipelineName <<- YAML

      - label: ":mac: MacOS Unit Tests"
        key: "macos-unit-tests-extended"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Filebeat: MacOS Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_X86_64}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":mac: MacOS ARM Unit Tests"
        key: "macos-arm64-unit-tests-extended"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Filebeat: MacOS ARM Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_ARM}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
YAML
fi

if are_conditions_met_arm_tests; then
  cat >> $pipelineName <<- YAML

      - label: ":linux: ARM Ubuntu Unit Tests"
        key: "extended-arm64-unit-test"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Filebeat: Unit Tests ARM"
        agents:
          provider: "aws"
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML

  - group: "Windows Extended Testing"
    key: "extended-tests-win"
    steps:
      - label: ":windows: Win 2019 Unit Tests"
        key: "windows-extended-2019"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Filebeat: Win-2019 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":windows:-11 Unit Tests"
        key: "windows-extended-11"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Filebeat: Win-11 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":windows:-10 Unit Tests"
        key: "windows-extended-10"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Filebeat: Win-10 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
YAML
fi

echo "Check and add the Packaging into the pipeline"
if are_conditions_met_packaging; then
cat >> $pipelineName <<- YAML

  - group: "Packaging"
    key: "packaging"
    depends_on:
      - "mandatory-tests"

    steps:
      - label: Package pipeline
        commands: ".buildkite/scripts/packaging/package-step.sh"
        notify:
        - github_commit_status:
            context: "Filebeat: Packaging"

YAML
fi

echo "+++ Printing dynamic steps"
cat $pipelineName | yq . -P

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
