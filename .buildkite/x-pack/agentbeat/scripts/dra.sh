set -uo pipefail

DRY_RUN=""

if [[ "${BUILDKITE_PULL_REQUEST:="false"}" != "false" || "${BUILDKITE_BRANCH}" == "ci_agentbeat_pack" ]]; then
    BRANCH=main
    # DRY_RUN="--dry-run"
    echo "+++ Running in PR and setting branch main and --dry-run"
fi

BEAT_VERSION=$(make get-version)
DRA_PROJECT_ID="beats/agentbeat"
DRA_PROJECT_ARTIFACT_ID="agentbeat"

CI_DRA_ROLE_PATH="kv/ci-shared/release/dra-role"

# TODO use common function
retry() {
  local retries=$1
  shift
  local count=0
  until "$@"; do
    exit=$?
    wait=$((2 ** count))
    count=$((count + 1))
    if [ $count -lt "$retries" ]; then
      >&2 echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      >&2 echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

function release_manager_login {
  DRA_CREDS_SECRET=$(retry 5 vault kv get -field=data -format=json ${CI_DRA_ROLE_PATH})
  VAULT_ADDR_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.vault_addr')
  VAULT_ROLE_ID_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.role_id')
  VAULT_SECRET=$(echo ${DRA_CREDS_SECRET} | jq -r '.secret_id')
  export VAULT_ADDR_SECRET VAULT_ROLE_ID_SECRET VAULT_SECRET
}

release_manager_login

echo "+++ :hammer_and_pick: Publishing $BRANCH $DRA_WORKFLOW DRA artifacts..."
cd x-pack/agentbeat && docker run --rm \
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
        --artifact-set "agentbeat" \
        ${DRY_RUN}