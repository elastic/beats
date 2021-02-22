#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# Start Elastic Agent
start_es_agent()
{
  if [ "$(pidof systemd && echo "systemd" || echo "other")" = "other" ]; then
    log "INFO" "[start_es_agent] starting Elastic Agent"
    sudo service elastic-agent start
    log "INFO" "[start_es_agent] Elastic Agent $STACK_VERSION started"
  else
    log "INFO" "[start_es_agent] enabling and starting Elastic Agent"
    sudo systemctl enable elastic-agent
    sudo systemctl start elastic-agent
    log "INFO" "[start_es_agent] Elastic Agent $STACK_VERSION started"
  fi
}

start_es_agent
