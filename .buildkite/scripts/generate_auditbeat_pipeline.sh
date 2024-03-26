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
        command:
          - ".buildkite/auditbeat/scripts/unit-tests.sh"
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
        command:
          - ".buildkite/auditbeat/scripts/unit-tests.sh"
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

      - label: ":windows:-{{matrix.image}} Unit Tests"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        notify:
          - github_commit_status:
              context: "Auditbeat: windows/Unit Tests"
        agents:
          provider: "gcp"
          image: "{{matrix.image}}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 200
          disk_type: "pd-ssd"
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_2016}"
              - "${IMAGE_WIN_2022}"
        artifact_paths:
          - "auditbeat/build/*.xml"
          - "auditbeat/build/*.json"

      - label: ":linux: Crosscompile"
        command:
          - ".buildkite/auditbeat/scripts/crosscompile.sh"
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
          command: ".buildkite/scripts/unit_tests.sh"
          agents:
            provider: "gcp"
            image: "${IMAGE_UBUNTU_ARM64}"
            machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
          artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
YAML
fi

if are_conditions_met_macos_tests; then
  cat >> $pipelineName <<- YAML
    - group: "MacOS Extended Testing"
      key: "extended-tests-macos"
      steps:
        - label: ":mac: MacOS Unit Tests"
          key: "extended-macos-unit-tests"
          command: ".buildkite/scripts/unit_tests.sh"
          agents:
            provider: "orka"
            imagePrefix: "${IMAGE_MACOS_X86_64}"
          artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML
    - group: "Windows Extended Testing"
      key: "extended-tests-win"
      steps:
        - label: ":windows: Windows 2019 Unit Tests"
          key: "extended-win-2019-unit-tests"
          command: ".buildkite/scripts/win_unit_tests.ps1"
          agents:
            provider: "gcp"
            image: "${IMAGE_WIN_2019}"
            machine_type: "${GCP_WIN_MACHINE_TYPE}"
            disk_size: 100
            disk_type: "pd-ssd"
          artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

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

YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
