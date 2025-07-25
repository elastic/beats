#!/usr/bin/env bash
set -euo pipefail

function ech_up() {
  echo "~~~ Starting ECH Stack"
  local WORKSPACE=$(git rev-parse --show-toplevel)
  local TF_DIR="${WORKSPACE}/testing/terraform-ech/"
  local STACK_VERSION=${1:?"Error: Specify stack version: ech_up [stack_version]"}
  local ECH_REGION=${2:-"gcp-us-west2"}


  BUILDKITE_BUILD_CREATOR="${BUILDKITE_BUILD_CREATOR:-"$(get_git_user_email)"}"
  BUILDKITE_BUILD_NUMBER="${BUILDKITE_BUILD_NUMBER:-"0"}"
  BUILDKITE_PIPELINE_SLUG="${BUILDKITE_PIPELINE_SLUG:-"beat-fips-ech-tests"}"

  pushd "${TF_DIR}"
  terraform init
  terraform apply \
    -auto-approve \
    -var="stack_version=${STACK_VERSION}" \
    -var="ech_region=${ECH_REGION}" \
    -var="creator=${BUILDKITE_BUILD_CREATOR}" \
    -var="buildkite_id=${BUILDKITE_BUILD_NUMBER}" \
    -var="pipeline=${BUILDKITE_PIPELINE_SLUG}"

  export ES_HOST=$(terraform output -raw es_host)
  export ES_USER=$(terraform output -raw es_username)
  export ES_PASS=$(terraform output -raw es_password)
  export KIBANA_HOST=$(terraform output -raw kibana_endpoint)
  export KIBANA_USER=$ES_USER
  export KIBANA_PASS=$ES_PASS
  popd
}

function ech_down() {
  echo "~~~ Tearing down the ECH Stack"
  local WORKSPACE=$(git rev-parse --show-toplevel)
  local TF_DIR="${WORKSPACE}/testing/terraform-ech/"

  pushd "${TF_DIR}"
  terraform init
  terraform destroy -auto-approve
  popd
}

function get_git_user_email() {
  if ! git rev-parse --is-inside-work-tree &>/dev/null; then
    echo "unknown"
    return
  fi

  local email
  email=$(git config --get user.email)

  if [ -z "$email" ]; then
    echo "unknown"
  else
    echo "$email"
  fi
}
