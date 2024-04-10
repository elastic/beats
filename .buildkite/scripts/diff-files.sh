#!/usr/bin/env bash
set -ueo pipefail
# This script is used to detect what diff command we are going to use to detect the changed files for triggering pipelines
if [[ -n "${BUILDKITE_PULL_REQUEST:-}" ]]; then
  git diff --name-only ${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-}...HEAD
else
  git diff --name-only HEAD~1
fi
