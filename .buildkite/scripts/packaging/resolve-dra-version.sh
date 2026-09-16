#!/usr/bin/env bash
##
##  Resolves STACK_VERSION, VERSION_QUALIFIER, and DRA_UPLOAD for the
##  current DRA workflow, so dra-prep-pipeline.yml can be uploaded with
##  them baked in as literal values.
##
##  Expects WORKFLOW ("snapshot" or "staging") and DRY_RUN as input env,
##  already set by the calling step.

set -euo pipefail

echo "--- Resolving DRA stack version"
VERSION=$(make get-version)
STACK_VERSION="${VERSION}"

# For staging, embed the qualifier (e.g. alpha1) into stack_version so the
# plugin publishes under e.g. 9.0.0-alpha1. Snapshot never carries a
# qualifier; the plugin auto-appends -SNAPSHOT for snapshot workflow.
if [[ "${WORKFLOW:-}" == "staging" ]]; then
  # shellcheck disable=SC1091
  source .buildkite/scripts/version_qualifier.sh
  if [[ -n "${VERSION_QUALIFIER:-}" ]]; then
    STACK_VERSION="${VERSION}-${VERSION_QUALIFIER}"
  fi
fi
export STACK_VERSION
export VERSION_QUALIFIER="${VERSION_QUALIFIER:-}"

# DRY_RUN env-var contract: when set to "true" from the Buildkite UI, the
# plugin runs but does not upload to GCS, and the annotate + processing-
# trigger steps are skipped (see dra-prep-pipeline.yml's `if:` conditions).
export DRA_UPLOAD="true"
if [[ "${DRY_RUN:-}" == "true" ]]; then
  export DRA_UPLOAD="false"
fi
