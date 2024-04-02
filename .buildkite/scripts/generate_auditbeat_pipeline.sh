#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.auditbeat-dynamic.yml"

# TODO: steps: must be always included
echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML

steps:
  - group: "Auditbeat Mandatory Testing"
    key: "mandatory-tests"

    steps:
      - label: ":ubuntu: Unit Tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: linux/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "auditbeat/build/*.xml"
          - "auditbeat/build/*.json"

      - label: ":rhel: Unit Tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: rhel/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_RHEL9}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "auditbeat/build/*.xml"
          - "auditbeat/build/*.json"

      - label: ":windows:-2016 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Auditbeat: windows 2016/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "auditbeat/build/*.xml"
          - "auditbeat/build/*.json"

      - label: ":windows:-2022 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "Auditbeat: windows 2022/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "auditbeat/build/*.xml"
          - "auditbeat/build/*.json"

      - label: ":linux: Crosscompile"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        env:
          GOX_FLAGS: "-arch amd64"
        notify:
          - github_commit_status:
              context: "Auditbeat: Crosscompile"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

echo "Check and add the Extended Tests into the pipeline"

if are_conditions_met_arm_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Extended Tests"
    key: "extended-tests-arm"
    steps:
      - label: ":arm: ARM64 Unit Tests"
        key: "extended-arm64-unit-tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_ARM_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
        notify:
          - github_commit_status:
              context: "Auditbeat: ARM Unit tests"
YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Windows Extended Testing"
    key: "extended-tests-win"
    steps:
      - label: ":windows: Windows 2019 Unit Tests"
        key: "extended-win-2019-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
        notify:
          - github_commit_status:
              context: "Auditbeat: Windows 2019 Unit Tests"

      - label: ":windows: Windows 10 Unit Tests"
        key: "extended-win-10-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
        notify:
          - github_commit_status:
              context: "Auditbeat: Windows 10 Unit Tests"

      - label: ":windows: Windows 11 Unit Tests"
        key: "extended-win-11-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
        notify:
          - github_commit_status:
              context: "Auditbeat: Windows 11 Unit Tests"
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
              context: "Auditbeat: Packaging"


YAML
fi

echo "+++ Printing dynamic steps"
cat $pipelineName | yq . -P

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
