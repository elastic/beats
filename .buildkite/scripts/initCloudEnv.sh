#!/usr/bin/env bash
set -euo pipefail

REPO_DIR=$(pwd)
AWS_SERVICE_ACCOUNT_SECRET_PATH="kv/ci-shared/platform-ingest/aws_account_auth"

exportAwsSecrets() {
  local awsSecretKey
  local awsAccessKey

  awsSecretKey=$(retry_with_count 5 vault kv get -field secret_key ${AWS_SERVICE_ACCOUNT_SECRET_PATH})
  awsAccessKey=$(retry_with_count 5 vault kv get -field access_key ${AWS_SERVICE_ACCOUNT_SECRET_PATH})

  echo "~~~ Exporting AWS secrets"
  export AWS_ACCESS_KEY_ID=$awsAccessKey
  export AWS_SECRET_ACCESS_KEY=$awsSecretKey
  export TEST_TAGS="${TEST_TAGS:+$TEST_TAGS,}aws"

  # AWS_REGION is not set here, since AWS region is taken from *.tf file:
  # - x-pack/metricbeat/module/aws/terraform.tf
  # - x-pack/filebeat/input/awscloudwatch/_meta/terraform/variables.tf
}

terraformApply() {
  TF_VAR_BRANCH=$(echo "${BUILDKITE_BRANCH}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
  TF_VAR_CREATED_DATE=$(date +%s)
  export TF_VAR_BUILD_ID="${BUILDKITE_BUILD_ID}"
  export TF_VAR_ENVIRONMENT="ci"
  export TF_VAR_REPO="${REPO}"
  export TF_VAR_BRANCH
  export TF_VAR_CREATED_DATE

  echo "Terraform Init on $MODULE_DIR"
  terraform -chdir="$MODULE_DIR" init

  echo "Terraform Apply on $MODULE_DIR"
  terraform -chdir="$MODULE_DIR" apply -auto-approve
}

terraformDestroy() {
  echo "--- Terraform Cleanup"
  cd $REPO_DIR
  find "$MODULE_DIR" -name terraform.tfstate -print0 | while IFS= read -r -d '' tfstate; do
    cd "$(dirname "$tfstate")"
    echo "Uploading terraform.tfstate to Buildkite artifacts"
    buildkite-agent artifact upload "**/terraform.tfstate"
    if ! terraform destroy -auto-approve; then
      return 1
    fi
  cd -
  done
  return 0
}

trap 'terraformDestroy' EXIT
exportAwsSecrets

max_retries=2
timeout=5
retries=0

while true; do
  echo "~~~ Setting up Terraform"
  out=$(terraformApply 2>&1)
  exit_code=$?

  echo "$out"

  if [ $exit_code -eq 0 ]; then
    break
  else
    retries=$((retries + 1))

    if [ $retries -gt $max_retries ]; then
      terraformDestroy
      echo "+++ Terraform init & apply failed: $out"
      exit 1
    fi

    terraformDestroy

    sleep_time=$((timeout * retries))
    echo "~~~~ Retry #$retries failed. Retrying after ${sleep_time}s..."
    sleep $sleep_time
  fi
done
