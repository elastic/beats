#!/usr/bin/env bash
##
##  Downloads Buildkite build artifacts, runs the beats-specific rename
##  (docker filename + dependencies CSV, from prepare-dra-artifacts.sh),
##  and stages the workflow's slice into artifacts/ for elastic/dra-prep.
##
##  On release branches, beats builds both snapshot and staging in the same
##  build, so filenames distinguish them: snapshot files contain "-SNAPSHOT-"
##  (or end in "-SNAPSHOT.csv"); staging files do not.
##
##  Runs in dra-prep-pipeline.yml, a separate pipeline from the one that
##  packaged the artifacts, so the download is scoped to the parent build
##  via BUILDKITE_TRIGGERED_FROM_BUILD_ID (auto-propagated by Buildkite to
##  any build started by a trigger step).
##

set -euo pipefail

WORKFLOW="${DRA_WORKFLOW:?DRA_WORKFLOW is required}"
PARENT_BUILD_ID="${BUILDKITE_TRIGGERED_FROM_BUILD_ID:?BUILDKITE_TRIGGERED_FROM_BUILD_ID is required}"

echo "--- Restoring artifacts from parent build ${PARENT_BUILD_ID}"
buildkite-agent artifact download "build/**/*" . --build "${PARENT_BUILD_ID}"

echo "--- Normalizing filenames (${WORKFLOW})"
.buildkite/scripts/packaging/prepare-dra-artifacts.sh "${WORKFLOW}"

echo "--- Preparing ${WORKFLOW} artifacts"
mkdir -p artifacts

if [[ "${WORKFLOW}" == "snapshot" ]]; then
  find build/distributions -type f \( -name "*-SNAPSHOT-*" -o -name "*-SNAPSHOT.csv" \) \
    -exec cp {} artifacts/ \;
else
  find build/distributions -type f ! -name "*-SNAPSHOT-*" ! -name "*-SNAPSHOT.csv" \
    -exec cp {} artifacts/ \;
fi

if ! ls artifacts/* >/dev/null 2>&1; then
  echo "ERROR: no ${WORKFLOW} artifacts found in artifacts/" >&2
  exit 1
fi

echo "Staged artifacts:"
ls -1 artifacts/
