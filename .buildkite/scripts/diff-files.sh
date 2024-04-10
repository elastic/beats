#!/usr/bin/env bash
set -ueo pipefail

if [[ -n "${BUILDKITE_PULL_REQUEST:-}" ]]; then
  git diff --name-only ${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-}...HEAD
else
  git diff --name-only HEAD~1
fi
