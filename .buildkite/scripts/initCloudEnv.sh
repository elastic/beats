#!/usr/bin/env bash
set -euo pipefail

REPO_DIR=$(pwd)
AWS_SERVICE_ACCOUNT_SECRET_PATH="kv/ci-shared/platform-ingest/aws_ingest_ci"

exportAwsSecrets() {
  local awsSecretKey
  local awsAccessKey

  awsSecretKey=$(retry -t 5 -- vault kv get -field secret_key "${AWS_SERVICE_ACCOUNT_SECRET_PATH}")
  awsAccessKey=$(retry -t 5 -- vault kv get -field access_key "${AWS_SERVICE_ACCOUNT_SECRET_PATH}")

  echo "~~~ Exporting AWS secrets"
  export AWS_ACCESS_KEY_ID=$awsAccessKey
  export AWS_SECRET_ACCESS_KEY=$awsSecretKey

  # AWS_REGION is not set here, since AWS region is taken from beat corresponding *.tf file:
  # - x-pack/metricbeat/module/aws/terraform.tf
  # - x-pack/filebeat/input/awscloudwatch/_meta/terraform/variables.tf
}

terraformApply() {
  echo "~~~ Exporting Terraform Env Vars"
  TF_VAR_BRANCH=$(echo "${BUILDKITE_BRANCH}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
  TF_VAR_CREATED_DATE=$(date +%s)
  export TF_VAR_BUILD_ID="${BUILDKITE_BUILD_ID}"
  export TF_VAR_ENVIRONMENT="ci"
  export TF_VAR_REPO="beats"
  export TF_VAR_BRANCH
  export TF_VAR_CREATED_DATE

  echo "~~~ Terraform Init on $MODULE_DIR"
  terraform -chdir="$MODULE_DIR" init

  echo "~~~ Terraform Apply on $MODULE_DIR"
  terraform -chdir="$MODULE_DIR" apply -auto-approve
}

terraformDestroy() {
  echo "~~~ Terraform Destroy"
  cd $REPO_DIR
  find "$MODULE_DIR" -name terraform.tfstate -print0 | while IFS= read -r -d '' tfstate; do
    cd "$(dirname "$tfstate")"
    buildkite-agent artifact upload "**/terraform.tfstate"
    buildkite-agent artifact upload "**/.terraform/**"
    buildkite-agent artifact upload "outputs*.yml"
    if ! terraform destroy -auto-approve; then
      return 1
    fi
    cd -
  done
  return 0
}

dockerUp() {
  echo "~~~ Run docker-compose services for emulated cloud env"
  docker-compose -f .buildkite/deploy/docker/docker-compose.yml up -d
}

dockerTeardown() {
  echo "~~~ Docker Compose Teardown"
  docker-compose -f .buildkite/deploy/docker/docker-compose.yml down -v
}

teardown() {
  terraformDestroy
  dockerTeardown
}

trap 'teardown' EXIT

exportAwsSecrets
dockerUp
terraformApply
