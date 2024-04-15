#!/usr/bin/env bash
set -euo pipefail

# What Terraform Module will run
if [[ "$BUILDKITE_PIPELINE_SLUG" == "beats-xpack-filebeat" ]]; then
  export MODULE_DIR="x-pack/filebeat/input/awss3/_meta/terraform"
fi

teardown() {
  popd
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
pushd "$MODULE_DIR"
export TF_VAR_BRANCH=$(echo "${BUILDKITE_BRANCH}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
export TF_VAR_BUILD_ID="${BUILDKITE_BUILD_ID}"
export TF_VAR_CREATED_DATE=$(date +%s)
export TF_VAR_ENVIRONMENT="ci"
export TF_VAR_REPO="${REPO}"
terraform init && terraform apply -auto-approve

# Run tests
echo "~~~ Run Cloud Tests for $BEATS_PROJECT_NAME"
pushd "${BEATS_PROJECT_NAME}"
mage build test
