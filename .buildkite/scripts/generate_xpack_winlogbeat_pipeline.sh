#!/usr/bin/env bash

source .buildkite/scripts/common.sh

set -euo pipefail

pipelineName="pipeline.xpack-winlogbeat-dynamic.yml"

echo "Add the mandatory and extended tests without additional conditions into the pipeline"
if are_conditions_met_mandatory_tests; then
  cat > $pipelineName <<- YAML

steps:

  - group: "Mandatory Tests"
    key: "mandatory-tests"
    steps:

      - label: ":windows: Windows 2019 Unit (MODULE) Tests"
<<<<<<< HEAD
        key: "mandatory-win-2019-unit-tests"
        command: ".buildkite/scripts/win_unit_tests.ps1"
=======
        key: "mandatory-win-2019-module-unit-tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
>>>>>>> c749dacac1 (Split windows steps (#38782))
        env:
          MODULE: $MODULE
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2019}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"

<<<<<<< HEAD
      - label: ":windows: Windows 2016/2022 Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        key: "mandatory-win-unit-tests"
=======
      - label: ":windows: Windows 2016 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        key: "mandatory-win-2016-unit-tests"
>>>>>>> c749dacac1 (Split windows steps (#38782))
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_2016}"
          machine_type: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
<<<<<<< HEAD
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_2016}"
              - "${IMAGE_WIN_2022}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
=======
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 2016 Unit Tests"

      - label: ":windows: Windows 2022 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
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
>>>>>>> c749dacac1 (Split windows steps (#38782))

# echo "Add the extended windows tests into the pipeline"
# TODO: ADD conditions from the main pipeline

  - group: "Extended Windows Tests"
    key: "extended-win-tests"
    steps:
<<<<<<< HEAD

      - label: ":windows: Windows Unit Tests - {{matrix.image}}"
        command: ".buildkite/scripts/win_unit_tests.ps1"
        key: "extended-win-unit-tests"
=======
      - label: ":windows: Windows 10 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
        key: "extended-win-10-unit-tests"
>>>>>>> c749dacac1 (Split windows steps (#38782))
        agents:
          provider: "gcp"
          image: "${IMAGE_WIN_10}"
          machineType: "${GCP_WIN_MACHINE_TYPE}"
          disk_size: 100
          disk_type: "pd-ssd"
<<<<<<< HEAD
        matrix:
          setup:
            image:
              - "${IMAGE_WIN_10}"
              - "${IMAGE_WIN_11}"
              - "${IMAGE_WIN_2019}"
        artifact_paths: "${BEATS_PROJECT_NAME}/build/*.*"
=======
        artifact_paths:
          - "$BEATS_PROJECT_NAME/build/*.xml"
          - "$BEATS_PROJECT_NAME/build/*.json"
        notify:
          - github_commit_status:
              context: "$BEATS_PROJECT_NAME: Windows 10 Unit Tests"

      - label: ":windows: Windows 11 Unit Tests"
        command: |
          Set-Location -Path $BEATS_PROJECT_NAME
          mage build unitTest
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
          mage build unitTest
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
>>>>>>> c749dacac1 (Split windows steps (#38782))

YAML
else
  echo "The conditions don't match to requirements for generating pipeline steps."
  exit 0
fi

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
          disk_size: 100
          disk_type: "pd-ssd"
        env:
          PLATFORMS: "${PACKAGING_PLATFORMS}"

YAML
fi

echo "--- Printing dynamic steps"     #TODO: remove if the pipeline is public
cat $pipelineName

echo "--- Loading dynamic steps"
buildkite-agent pipeline upload $pipelineName
