#!/usr/bin/env bash
##
##  Generates and uploads the DRA sub-pipeline for a given workflow (snapshot
##  or staging). Reads the version from `make get-version`, resolves the
##  staging qualifier via version_qualifier.sh, and emits a step that runs
##  elastic/dra-prep + (on non-dry-run builds) a summary annotator and the
##  unified-release-dra-processing trigger.
##
##  This script is invoked once per workflow from packaging.pipeline.yml
##  and leverages elastic/dra-prep-buildkite-plugin to upload DRA artifacts
##

set -eo pipefail

# snapshot or staging
WORKFLOW="$1"

VERSION=$(make get-version)
STACK_VERSION="${VERSION}"

# For staging, embed the qualifier (e.g. alpha1) into stack_version so the
# plugin publishes under e.g. 9.0.0-alpha1. Snapshot never carries a qualifier.
# The plugin auto-appends -SNAPSHOT for snapshot workflow.
if [[ "${WORKFLOW}" == "staging" ]]; then
  # shellcheck disable=SC1091
  source .buildkite/scripts/version_qualifier.sh
  if [[ -n "${VERSION_QUALIFIER:-}" ]]; then
    STACK_VERSION="${VERSION}-${VERSION_QUALIFIER}"
  fi
fi

# DRY_RUN env-var contract: when set to "true" from
# the Buildkite UI, the plugin runs but does not upload to GCS, and we skip
# both the annotator (summary URL would 404) and the processing trigger.
DRA_UPLOAD=true
if [[ "${DRY_RUN:-}" == "true" ]]; then
  DRA_UPLOAD=false
fi

echo "--- :arrow_right: DRA context"
echo "BUILDKITE_BRANCH=${BUILDKITE_BRANCH}"
echo "BUILDKITE_COMMIT=${BUILDKITE_COMMIT}"
echo "WORKFLOW=${WORKFLOW}"
echo "STACK_VERSION=${STACK_VERSION}"
echo "DRA_UPLOAD=${DRA_UPLOAD}"

trigger_step=""
annotate_step=""
if [[ "${DRA_UPLOAD}" == "true" ]]; then
  trigger_step=$(cat <<TRIG

  - label: ":pipeline: Trigger DRA processing (${WORKFLOW})"
    trigger: "unified-release-dra-processing"
    depends_on: "dra-prep-${WORKFLOW}"
    build:
      env:
        DRA_PRODUCT_ID: "beats"
        DRA_STACK_VERSION: "${STACK_VERSION}"
        DRA_WORKFLOW: "${WORKFLOW}"
TRIG
)
  annotate_step=$(cat <<ANN

  - label: ":memo: Annotate DRA summary (${WORKFLOW})"
    key: "dra-annotate-${WORKFLOW}"
    depends_on: "dra-prep-${WORKFLOW}"
    command: |
      set -euo pipefail
      buildkite-agent artifact download "artifacts/dra/beats/*/manifest-*.json" . --step "dra-prep-${WORKFLOW}"
      manifest=\$(find artifacts/dra/beats -name "manifest-*.json" | head -1)
      build_id=\$(jq -r '.build_id' "\${manifest}")
      version=\$(jq -r '.version' "\${manifest}")
      url="https://artifacts-${WORKFLOW}.elastic.co/dra-builds/\${BUILDKITE_PIPELINE_SLUG}/\${BUILDKITE_BUILD_NUMBER}/\${build_id}/summary-\${version}.html"
      printf "**${WORKFLOW} summary link:** [%s](%s)\n" "\${url}" "\${url}" | buildkite-agent annotate --style=success --append
    agents:
      provider: gcp
      image: "${IMAGE_UBUNTU_X86_64}"
    timeout_in_minutes: 5
ANN
)
fi

echo "--- Generating DRA sub-pipeline for ${WORKFLOW} (upload=${DRA_UPLOAD})"
cat <<PIPELINE | buildkite-agent pipeline upload
steps:
  - label: ":package: DRA Prep (${WORKFLOW})"
    key: "dra-prep-${WORKFLOW}"
    env:
      DRA_WORKFLOW: "${WORKFLOW}"
      VERSION_QUALIFIER: "${VERSION_QUALIFIER:-}"
    command: ".buildkite/scripts/stage-dra-artifacts.sh"
    agents:
      provider: gcp
      image: "${IMAGE_UBUNTU_X86_64}"
      machineType: "${GCP_DEFAULT_MACHINE_TYPE}"
    timeout_in_minutes: 30
    artifact_paths:
      - "artifacts/dra/beats/*/manifest-*.json"
    plugins:
      - elastic/dra-prep#v0.1.5:
          product_id: "beats"
          stack_version: "${STACK_VERSION}"
          workflow: "${WORKFLOW}"
          upload: ${DRA_UPLOAD}
${annotate_step}
${trigger_step}
PIPELINE
