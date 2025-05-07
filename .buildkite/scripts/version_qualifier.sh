#!/usr/bin/env bash

# An opinionated approach to managing the Elastic Qualifier for the DRA in a Google Bucket
# instead of using a Buildkite env variable.

if [[ -n "$VERSION_QUALIFIER" ]]; then
  echo "~~~ VERSION_QUALIFIER externally set to [$VERSION_QUALIFIER]"
  return 0
fi

# DRA_BRANCH can be used for manually testing packaging with PRs
# e.g. define `DRA_BRANCH="main"` under Options/Environment Variables in the Buildkite UI after clicking new Build
BRANCH="${DRA_BRANCH:="${BUILDKITE_BRANCH:=""}"}"

qualifier=""
URL="https://storage.googleapis.com/dra-qualifier/${BRANCH}"
if curl -sf -o /dev/null "$URL" ; then
  qualifier=$(curl -s "$URL")
fi

export VERSION_QUALIFIER="$qualifier"
echo "~~~ VERSION_QUALIFIER set to [$VERSION_QUALIFIER]"
