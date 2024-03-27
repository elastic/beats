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
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: linux/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":rhel: Unit Tests"
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: rhel/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_RHEL9}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows:-2016 Unit Tests"
        key: "windows-2016"
        command: "mage -d ${BEATS_PROJECT_NAME} unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: windows-2016/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows:-2022 Unit Tests"
        key: "windows-2022"
        command: "mage -d ${BEATS_PROJECT_NAME} unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: windows-2022/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":linux: Crosscompile"
        command:
          - "make -C auditbeat crosscompile"
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
              context: "Auditbeat: MacOS Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_X86_64}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":mac: MacOS ARM Unit Tests"
        key: "macos-arm64-unit-tests-extended"
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: MacOS ARM Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_ARM}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

YAML
fi

if are_conditions_met_arm_tests; then
  cat >> $pipelineName <<- YAML
      - label: ":linux: ARM Ubuntu Unit Tests"
        key: "extended-arm64-unit-test"
        command: "cd ${BEATS_PROJECT_NAME} && mage unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: Unit Tests ARM"
        agents:
          provider: "aws"
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Windows Extended Testing"
    key: "extended-tests-win"

    steps:
      - label: ":windows:-2019 Unit Tests"
        key: "windows-2019-extended"
        command: "mage -d ${BEATS_PROJECT_NAME} unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: Win-2019 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows:-11 Unit Tests"
        key: "windows-11-extended"
        command: "mage -d ${BEATS_PROJECT_NAME} unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: Win-11 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_11}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      - label: ":windows:-10 Unit Tests"
        key: "windows-10-extended"
        command: "mage -d ${BEATS_PROJECT_NAME} unitTest"
        notify:
          - github_commit_status:
              context: "Auditbeat: Win-10 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
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

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
