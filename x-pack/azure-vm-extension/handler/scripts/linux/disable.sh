#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

log "Stopping Elastic Agent" "INFO"
sudo service elastic-agent stop
log "Elastic Agent is stopped" "INFO"
