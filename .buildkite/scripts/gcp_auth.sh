#!/usr/bin/env bash

set -euo pipefail

echo "~~~ Authenticating GCP"
# Secrets must be redacted
# https://buildkite.com/docs/pipelines/managing-log-output#redacted-environment-variables
PRIVATE_CI_GCS_CREDENTIALS_PATH="kv/ci-shared/platform-ingest/gcp-platform-ingest-ci-service-account"
PRIVATE_CI_GCS_CREDENTIALS_SECRET=$(vault kv get -field plaintext -format=json ${PRIVATE_CI_GCS_CREDENTIALS_PATH})
export PRIVATE_CI_GCS_CREDENTIALS_SECRET
echo "${PRIVATE_CI_GCS_CREDENTIALS_SECRET}" > ./gcp.json
GOOGLE_APPLICATION_CREDENTIALS=$(realpath ./gcp.json)
export GOOGLE_APPLICATION_CREDENTIALS
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"