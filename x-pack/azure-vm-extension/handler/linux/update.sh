#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

log "INFO" "[Update_ElasticAgent] set update environment variable"
set_update_env_var
