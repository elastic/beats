#!/usr/bin/env bash
set -euo pipefail

## TODO: Set to empty string when Jenkins is disabled
if [[ "$DRY_RUN" == "false" ]]; then echo "--- Running in publish mode"; DRY_RUN=""; else echo "--- Running in dry-run mode"; DRY_RUN="--dry-run"; fi
BRANCH="${BUILDKITE_BRANCH}"

if [[ "${BUILDKITE_PULL_REQUEST:="false"}" != "false" ]]; then
    BRANCH=main
    DRY_RUN="--dry-run"
    echo "+++ Running in PR and setting branch main and --dry-run"
fi

BEAT_VERSION=$(make get-version)

CI_DRA_ROLE_PATH="kv/ci-shared/release/dra-role"

function release_manager_login {
  DRA_CREDS_SECRET=$(retry -t 5 -- vault kv get -field=data -format=json ${CI_DRA_ROLE_PATH})
  VAULT_ADDR_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.vault_addr')
  VAULT_ROLE_ID_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.role_id')
  VAULT_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.secret_id')
  export VAULT_ADDR_SECRET VAULT_ROLE_ID_SECRET VAULT_SECRET
}

release_manager_login

echo "+++ Changing permissions for the BK API commands"
sudo chown -R :1000 build/distributions/

echo "+++ :hammer_and_pick: Publishing $BRANCH $DRA_WORKFLOW DRA artifacts..."
docker run --rm \
        --name release-manager \
        -e VAULT_ADDR="${VAULT_ADDR_SECRET}" \
        -e VAULT_ROLE_ID="${VAULT_ROLE_ID_SECRET}" \
        -e VAULT_SECRET_ID="${VAULT_SECRET}" \
        --mount type=bind,readonly=false,src="${PWD}",target=/artifacts \
        docker.elastic.co/infra/release-manager:latest \
        cli collect \
        --project "beats" \
        --branch "${BRANCH}" \
        --commit "${BUILDKITE_COMMIT}" \
        --workflow "${DRA_WORKFLOW}" \
        --version "${BEAT_VERSION}" \
        --artifact-set "main" \
        ${DRY_RUN}