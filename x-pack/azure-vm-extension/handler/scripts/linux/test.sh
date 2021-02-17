#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh



test_this () {
  get_cloud_stack_version
  echo $STACK_VERSION
}

test_this
