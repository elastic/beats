#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# Stop Elastic Agent
Stop_ElasticAgent()
{
  if [ "$(pidof systemd && echo "systemd" || echo "other")" = "other" ]; then
    log "INFO" "[Stop_ElasticAgent] stopping Elastic Agent"
    sudo service elastic-agent stop
    log "INFO" "[Stop_ElasticAgent] Elastic Agent stopped"
  else
    log "INFO" "[Stop_ElasticAgent] stopping  Elastic Agent"
    sudo systemctl stop elastic-agent
    log "INFO" "[Stop_ElasticAgent] Elastic Agent stopped"
  fi
}

retry_backoff Stop_ElasticAgent
