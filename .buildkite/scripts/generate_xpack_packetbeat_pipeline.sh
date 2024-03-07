#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.xpack-metricbeat-dynamic.yml"

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
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

      - label: ":linux: Ubuntu System Tests"
        key: "mandatory-linux-system-test"
        command: "cd $BEATS_PROJECT_NAME && mage systemTest"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.xml"

      - label: ":rhel: RHEL-9 Unit Tests"
        key: "mandatory-rhel9-unit-test"
        command: ".buildkite/scripts/unit_tests.sh"
        agents:
          provider: "gcp"
          image: "${IMAGE_RHEL9_X86_64}"
          machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"


      - label: ":windows: Windows Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1"
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
              - "platform-ingest-beats-windows-2016-1709815915"
              - "platform-ingest-beats-windows-2022-1709811171"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      ##  TODO: uncomment when the issue https://github.com/elastic/beats/issues/38142 is solved
      # - label: ":windows: Windows 2022 System Tests"
      #   key: "mandatory-win-2022-system-tests"
      #   command: ".buildkite/scripts/win_unit_tests.ps1 systemtest"
      #   agents:
      #     provider: "gcp"
      #     image: "${IMAGE_WIN_2022}"
      #     machineType: "${GCP_WIN_MACHINE_TYPE}"
      #     disk_size: 100
      #     disk_type: "pd-ssd"
      #   artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

## TODO: this condition will be changed in the Phase 3 of the Migration Plan https://docs.google.com/document/d/1IPNprVtcnHlem-uyGZM0zGzhfUuFAh4LeSl9JFHMSZQ/edit#heading=h.sltz78yy249h

  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:

      - label: ":windows: Windows Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1"
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
              - "platform-ingest-beats-windows-10-1709818219"
              - "${IMAGE_WIN_11}"
              - "platform-ingest-beats-windows-2019-1709818219"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

      ##  TODO: uncomment when the issue https://github.com/elastic/beats/issues/38142 is solved
      # - label: ":windows: Windows 10 System Tests"
      #   key: "extended-win-10-system-tests"
      #   command: ".buildkite/scripts/win_unit_tests.ps1 systemtest"
      #   agents:
      #     provider: "gcp"
      #     image: "${IMAGE_WIN_10}"
      #     machineType: "${GCP_WIN_MACHINE_TYPE}"
      #     disk_size: 100
      #     disk_type: "pd-ssd"
      #   artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

# if [[ are_conditions_met_arm_tests || are_conditions_met_macos_tests ]]; then
#   cat >> $pipelineName <<- YAML

#   - group: "Extended Tests"
#     key: "extended-tests"
#     steps:

# YAML
# fi

# if  are_conditions_met_macos_tests; then
#   cat >> $pipelineName <<- YAML

#       - label: ":mac: MacOS Unit Tests"
#         key: "extended-macos-unit-tests"
#         command: ".buildkite/scripts/unit_tests.sh"
#         agents:
#           provider: "orka"
#           imagePrefix: "${IMAGE_MACOS_X86_64}"
#         artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

# YAML
# fi

# if  are_conditions_met_arm_tests; then
#   cat >> $pipelineName <<- YAML
#       - label: ":linux: ARM Ubuntu Unit Tests"
#         key: "extended-arm64-unit-test"
#         command: "cd $BEATS_PROJECT_NAME && mage build unitTest"
#         agents:
#           provider: "aws"
#           imagePrefix: "${IMAGE_UBUNTU_ARM_64}"
#           instanceType: "${AWS_ARM_INSTANCE_TYPE}"
#         artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

# YAML
# fi

echo "Check and add the Packaging into the pipeline"
if are_conditions_met_packaging; then
  cat >> $pipelineName <<- YAML

  - wait: ~
    depends_on:
      - step: "mandatory-tests"
        allow_failure: false

  - group: "Packaging"    # TODO: check conditions for future the main pipeline migration: https://github.com/elastic/beats/pull/28589
    key: "packaging"
    steps:
      - label: ":linux: Packaging Linux"
        key: "packaging-linux"
        command: "cd $BEATS_PROJECT_NAME && mage package"
        agents:
          provider: "gcp"
          image: "${IMAGE_UBUNTU_X86_64}"
          machineType: "${GCP_HI_PERF_MACHINE_TYPE}"
        env:
          PLATFORMS: "${PACKAGING_PLATFORMS}"

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

YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
