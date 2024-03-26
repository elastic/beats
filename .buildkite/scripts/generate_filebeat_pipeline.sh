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
        command:
          - ".buildkite/filebeat/scripts/unit-tests.sh"
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
        command:
          - ".buildkite/filebeat/scripts/integration-gotests.sh"
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
        command:
          - ".buildkite/filebeat/scripts/integration-pytests.sh"
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

      - label: ":windows:-{{matrix.image}} Unit Tests"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        notify:
          - github_commit_status:
              context: "Filebeat: windows/Unit Tests"
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
          - "filebeat/build/*.xml"
          - "filebeat/build/*.json"

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

if are_conditions_met_macos_tests; then
  cat >> $pipelineName <<- YAML
  - group: "MacOS Extended Testing"
    key: "extended-tests-macos"
    steps:
      - label: ":mac: MacOS Unit Tests"
        key: "macos-extended"
        if: build.env("GITHUB_PR_TRIGGER_COMMENT") == "filebeat for macos" || build.env("GITHUB_PR_LABELS") =~ /.*macOS.*/
        command:
          - ".buildkite/filebeat/scripts/unit-tests.sh"
        notify:
          - github_commit_status:
              context: "Filebeat/Extended: MacOS Unit Tests"
        agents:
          provider: "orka"
          imagePrefix: "${IMAGE_MACOS_X86_64}"
        artifact_paths: "filebeat/build/*.xml"
YAML
fi

if are_conditions_met_win_tests; then
  cat >> $pipelineName <<- YAML
  - group: "Windows Extended Testing"
    key: "extended-tests-win"
    steps:
    - label: ":windows: Win 2019 Unit Tests"
      key: "win-extended-2019"
      command: ".buildkite/scripts/win_unit_tests.ps1"
      notify:
        - github_commit_status:
            context: "Filebeat/Extended: Win-2019 Unit Tests"
      agents:
        provider: "gcp"
        image: "${IMAGE_WIN_2019}"
        machine_type: "${GCP_WIN_MACHINE_TYPE}"
        disk_size: 200
        disk_type: "pd-ssd"
      artifact_paths:
        - "filebeat/build/*.xml"
        - "filebeat/build/*.json"
YAML
fi

echo "Check and add the Packaging into the pipeline"
if are_conditions_met_packaging; then
cat >> $pipelineName <<- YAML
  - group: "Packaging"
    key: "packaging"
    if: build.env("BUILDKITE_PULL_REQUEST") != "false"
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
