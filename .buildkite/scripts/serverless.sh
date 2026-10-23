#!/usr/bin/env bash
set -euo pipefail

function serverless_up() {
  echo "~~~ Starting Serverless Project"
  local WORKSPACE TF_DIR
  WORKSPACE=$(git rev-parse --show-toplevel)
  TF_DIR="${WORKSPACE}/testing/terraform-serverless/"

  pushd "${TF_DIR}"
  terraform init
  terraform apply -auto-approve

  # Assign before exporting, so a failing terraform output fails the script.
  ES_HOST=$(terraform output -raw es_host)
  ES_USER=$(terraform output -raw es_username)
  ES_PASS=$(terraform output -raw es_password)
  KIBANA_HOST=$(terraform output -raw kibana_endpoint)
  export ES_HOST ES_USER ES_PASS KIBANA_HOST
  export KIBANA_USER=$ES_USER
  export KIBANA_PASS=$ES_PASS
  popd

  # The Beats under test write with an API key, as most serverless users do.
  # It's deleted with the project, so serverless_down doesn't need to revoke it.
  echo "~~~ Creating an Elasticsearch API key"
  ES_API_KEY=$(create_api_key)
  export ES_API_KEY
}

# create_api_key prints an API key in the id:api_key form the Beats expect.
function create_api_key() {
  curl -fsS -u "${ES_USER}:${ES_PASS}" -X POST "${ES_HOST}/_security/api_key" \
    -H 'Content-Type: application/json' \
    -d '{"name": "beats-ci", "expiration": "1d"}' |
    jq -r '"\(.id):\(.api_key)"'
}

function serverless_down() {
  echo "~~~ Tearing down the Serverless Project"
  local WORKSPACE TF_DIR
  WORKSPACE=$(git rev-parse --show-toplevel)
  TF_DIR="${WORKSPACE}/testing/terraform-serverless/"

  pushd "${TF_DIR}"
  terraform init
  terraform destroy -auto-approve
  popd
}
