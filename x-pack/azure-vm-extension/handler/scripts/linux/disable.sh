#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# Stop Elastic Agent
stop_es_agent()
{
  if [ "$(pidof systemd && echo "systemd" || echo "other")" = "other" ]; then
    log "INFO" "[stop_es_agent] stopping Elastic Agent"
    sudo service elastic-agent stop
    log "INFO" "[stop_es_agent] Elastic Agent stopped"
  else
    log "INFO" "[stop_es_agent] stopping  Elastic Agent"
    sudo systemctl stop elastic-agent
    log "INFO" "[stop_es_agent] Elastic Agent stopped"
  fi
}

stop_es_agent
