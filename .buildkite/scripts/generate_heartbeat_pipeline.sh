#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.heartbeat-dynamic.yml"

# TODO: steps: must be always included 
echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML
steps:
  - group: "Heartbeat Mandatory Testing"
    key: "mandatory-tests"    

    steps:
      
      - label: ":linux: Unit Tests / Ubuntu"
        command:
          - ".buildkite/heartbeat/scripts/unit-tests.sh"
        notify:
          - github_commit_status:
              context: "Heartbeat: Ubuntu/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"        
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"

      - label: ":linux: Unit Tests / RHEL"
        command:
          - ".buildkite/heartbeat/scripts/unit-tests.sh"
        notify:
          - github_commit_status:
              context: "Heartbeat: RHEL/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_RHEL9}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"        
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"          

      - label: ":windows: Unit Tests / Win 2016"
        command:
          - ".buildkite/scripts/win_unit_tests.ps1"
        notify:
          - github_commit_status:
              context: "Heartbeat: windows 2016/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_type: "pd-ssd"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"

      - label: ":windows: Unit Tests / Win 2016"
        command:
          - ".buildkite/scripts/win_unit_tests.ps1"
        notify:
          - github_commit_status:
              context: "Heartbeat: windows 2016/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_type: "pd-ssd"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"

      - label: ":windows: Unit Tests / Win 2022"
        command:
          - ".buildkite/scripts/win_unit_tests.ps1"
        notify:
          - github_commit_status:
              context: "Heartbeat: windows 2022/Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2022}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_type: "pd-ssd"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"          

      - label: ":ubuntu: Go Integration Tests"
        command:
          - ".buildkite/heartbeat/scripts/integration-gotests.sh"
        notify:
          - github_commit_status:
              context: "Heartbeat: Go Integration Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"

      - label: ":ubuntu: Python Integration Tests"
        command:
          - ".buildkite/heartbeat/scripts/integration-pytests.sh"
        notify:
          - github_commit_status:
              context: "Heartbeat: Python Integration Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"

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
      - label: ":linux: ARM64 Unit Tests"
        key: "arm-extended"        
        command:
          - ".buildkite/heartbeat/scripts/unit-tests.sh"
        notify:
          - github_commit_status:
              context: "Heartbeat/Extended: Unit Tests ARM"
        agents:
          provider: "aws"
          imagePrefix: "${AWS_IMAGE_UBUNTU_ARM_64}"
          instanceType: "${AWS_ARM_INSTANCE_TYPE}"
        artifact_paths: "heartbeat/build/*.xml"
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
              context: "Heartbeat/Extended: Win 2019 Unit Tests"
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_type: "pd-ssd"
        artifact_paths:
          - "heartbeat/build/*.xml"
          - "heartbeat/build/*.json"
      
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
              context: "Heatbeat: Packaging"


YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
