#!/usr/bin/env bash
##
##  Publishes the result of a beats-tests build as a GitHub commit status
##  on every commit the build covered: the tested commit and its
##  first-parent ancestors, back to the first one that another build of a
##  different commit already marked. If none is found within MAX_COMMITS,
##  only the tested commit is marked.
##
##  Merges to main and release branches do not run the tests, so without
##  this a failure only shows on the one commit the daily build tested.
##
##  Invoked from beats-tests-pipeline.yml once every trigger step has
##  finished. Set DRY_RUN=true to print the statuses without publishing.
##

set -euo pipefail

PIPELINE_FILE=.buildkite/beats-tests-pipeline.yml
STATUS_CONTEXT="beats-tests (daily)"
GITHUB_REPO=elastic/beats
MAX_COMMITS=100
DRY_RUN="${DRY_RUN:-false}"

if [[ ! "${BUILDKITE_COMMIT}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "BUILDKITE_COMMIT must be a full commit SHA, got [${BUILDKITE_COMMIT}]" >&2
  exit 1
fi

echo "--- Collecting the outcome of each trigger step"
mapfile -t keys < <(sed -n -E 's/^[[:space:]]+key:[[:space:]]*"?(test-[A-Za-z0-9_-]+)"?[[:space:]]*$/\1/p' "${PIPELINE_FILE}")
triggers=$(grep -c -E '^[[:space:]]*(- )?trigger:' "${PIPELINE_FILE}")
if ((${#keys[@]} != triggers)); then
  echo "${PIPELINE_FILE} has ${triggers} trigger steps but ${#keys[@]} \"test-\" keys; every trigger step needs one" >&2
  exit 1
fi

failed=""
errored=""
for key in "${keys[@]}"; do
  outcome=$(buildkite-agent step get outcome --step "${key}")
  echo "${key}: ${outcome}"
  case "${outcome}" in
    passed | soft_failed) ;;
    hard_failed) failed+="${failed:+, }${key#test-}" ;;
    *) errored+="${errored:+, }${key#test-}" ;;
  esac
done

state=success
summary=passed
if [[ -n "${errored}" ]]; then
  state=error
  summary="errored: ${errored}"
fi
if [[ -n "${failed}" ]]; then
  state=failure
  summary="failed: ${failed}${errored:+; errored: ${errored}}"
fi
description="Daily run #${BUILDKITE_BUILD_NUMBER} on ${BUILDKITE_COMMIT:0:10} ${summary}"
# GitHub rejects descriptions longer than 140 characters.
if ((${#description} > 140)); then
  description="${description:0:137}..."
fi

echo "--- Finding the commits covered by this build"
# shellcheck disable=SC2016 # GraphQL variables, not shell ones
query='
query($owner: String!, $name: String!, $oid: GitObjectID!, $first: Int!, $context: String!) {
  repository(owner: $owner, name: $name) {
    object(oid: $oid) {
      ... on Commit {
        history(first: $first) {
          nodes {
            oid
            messageHeadline
            parents(first: 1) { nodes { oid } }
            status { context(name: $context) { targetUrl } }
          }
        }
      }
    }
  }
}'
# A status already on the tested commit comes from an earlier build of the
# same commit, so the ancestors that build marked belong to this range too.
# shellcheck disable=SC2016 # jq variables, not shell ones
range='
.data.repository.object.history.nodes
| INDEX(.oid) as $c
| $c[$head].status.context.targetUrl as $previous
| [$head | recurse($c[.].parents.nodes[0].oid // empty | select($c[.]))] as $chain
| ($chain[1:] | map($c[.].status.context.targetUrl | . != null and . != $self and . != $previous) | index(true)) as $i
| {
    previous: $previous,
    bounded: ($i != null),
    commits: ($chain[:($i // 0) + 1] | map({sha: ., headline: $c[.].messageHeadline}))
  }'
covered=$(gh api graphql \
  -f query="${query}" \
  -f owner="${GITHUB_REPO%%/*}" \
  -f name="${GITHUB_REPO#*/}" \
  -f oid="${BUILDKITE_COMMIT}" \
  -F first="${MAX_COMMITS}" \
  -f context="${STATUS_CONTEXT}" |
  jq --arg head "${BUILDKITE_COMMIT}" --arg self "${BUILDKITE_BUILD_URL}" "${range}")

# Retrying the publish step of an old build must not replace a newer result.
previous=$(jq -r '.previous // empty' <<<"${covered}")
if [[ "${previous}" =~ ^"${BUILDKITE_BUILD_URL%/*}/"([0-9]+)$ ]] && ((BASH_REMATCH[1] > BUILDKITE_BUILD_NUMBER)); then
  echo "${BUILDKITE_COMMIT} already has the result of a newer build, ${previous}; not publishing"
  exit 0
fi

if [[ "$(jq -r .bounded <<<"${covered}")" != "true" ]]; then
  echo "No commit within the last ${MAX_COMMITS} has a \"${STATUS_CONTEXT}\" status from another build; only marking ${BUILDKITE_COMMIT}"
fi
jq -r '.commits[] | "\(.sha[0:10]) \(.headline)"' <<<"${covered}"

echo "--- Publishing \"${STATUS_CONTEXT}\" as ${state}: ${description}"
if [[ "${DRY_RUN}" == "true" ]]; then
  echo "DRY_RUN=true, not publishing"
  exit 0
fi
# Oldest first: if a request fails, the tested commit stays unmarked, so
# the next build does not stop there and covers the commits left behind.
for sha in $(jq -r '.commits | reverse | .[].sha' <<<"${covered}"); do
  gh api --method POST "repos/${GITHUB_REPO}/statuses/${sha}" \
    -f state="${state}" \
    -f context="${STATUS_CONTEXT}" \
    -f target_url="${BUILDKITE_BUILD_URL}" \
    -f description="${description}" \
    --silent
done
