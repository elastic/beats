#!/usr/bin/env bash
set -euo pipefail


REPO_DIR=$(pwd)

teardown() {
  # reset the directory to the root of the project
  cd $REPO_DIR
  # Teardown resources after using them
  echo "~~~ Terraform Cleanup"
  tf_cleanup "${MODULE_DIR}"              #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK

  echo "~~~ Docker Compose Cleanup"
  docker-compose -f .ci/jobs/docker-compose.yml down -v         #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK
}

tf_cleanup() {
  DIRECTORY=${1:-.}

  for tfstate in $(find $DIRECTORY -name terraform.tfstate); do
    cd $(dirname $tfstate)
    terraform init
    if ! terraform destroy -auto-approve; then
        echo "+++ Failed to Terraform destroy the resources"
    fi
    cd -
  done
}

trap 'teardown' EXIT

# Prepare the cloud resources using Terraform
#startCloudTestEnv "${MODULE_DIR}"
echo "~~~ Loading creds"
set +o xtrace
export AWS_ACCESS_KEY_ID=$BEATS_AWS_ACCESS_KEY
export AWS_SECRET_ACCESS_KEY=$BEATS_AWS_SECRET_KEY
export TEST_TAGS="${TEST_TAGS:+$TEST_TAGS,}aws"
set -o xtrace

echo "~~~ Run docker-compose services for emulated cloud env"
docker-compose -f .ci/jobs/docker-compose.yml up -d        #TODO: move all docker-compose files from the .ci to .buildkite folder before switching to BK
echo "~~~ Initialize TF cloud resources"
cd "$MODULE_DIR"
export TF_VAR_BRANCH=$(echo "${BUILDKITE_BRANCH}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
export TF_VAR_BUILD_ID="${BUILDKITE_BUILD_ID}"
export TF_VAR_CREATED_DATE=$(date +%s)
export TF_VAR_ENVIRONMENT="ci"
export TF_VAR_REPO="${REPO}"
terraform init && terraform apply -auto-approve
cd -

# Run tests
echo "~~~ Run Cloud Tests for $BEATS_PROJECT_NAME"
cd "${BEATS_PROJECT_NAME}"
mage build test
