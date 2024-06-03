#!/usr/bin/env bash

set -exuo pipefail

DIRECTORY=${1:-.}

FAILED=0
for tfstate in $(find $DIRECTORY -name terraform.tfstate); do
  cd $(dirname $tfstate)
  terraform init
  if ! terraform destroy -auto-approve; then
    FAILED=1
  fi
  cd -
done

exit $FAILED
