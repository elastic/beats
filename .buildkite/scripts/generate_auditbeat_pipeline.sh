#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.auditbeat-dynamic.yml"

# TODO: steps: must be always included
echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML

steps:
  - group: "Mandatory Testing"
    key: "mandatory-tests"

    steps:
      - label: ":ubuntu: Ubuntu Unit Tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Ubuntu Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":rhel: RHEL Unit Tests"
        command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: RHEL9 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_RHEL9}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":windows: Windows 2016 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2016 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":windows: Windows 2022 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2022 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":linux: Crosscompile"
        command: "make -C $BEATS_PROJECT_NAME crosscompile"
        env:
          GOX_FLAGS: "-arch amd64"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Crosscompile"
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
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: MacOS Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_X86_64}"
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"

      - label: ":mac: MacOS ARM Unit Tests"
        key: "macos-arm64-unit-tests-extended"
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: MacOS ARM Unit Tests"
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
      - label: ":linux: Ubuntu ARM Unit Tests"
        key: "extended-arm64-unit-test"
        command: "cd ${BEATS_PROJECT_NAME} && mage build unitTest"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Ubuntu ARM Unit Tests"
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
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2019 Unit Tests"

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
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 10 Unit Tests"

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
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 11 Unit Tests"
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
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Packaging Linux"

      - label: ":linux: Packaging ARM"
        key: "packaging-arm"
        command: "cd $BEATS_PROJECT_NAME && mage package"
        agents:
          provider: "aws"
          imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        env:
          PLATFORMS: "${PACKAGING_ARM_PLATFORMS}"
          PACKAGES: "docker"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Packaging Linux ARM"


YAML
fi

echo "+++ Printing dynamic steps"
cat $pipelineName | yq . -P

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName