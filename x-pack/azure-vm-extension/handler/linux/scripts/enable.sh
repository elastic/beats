#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# for status
name="Enable elastic agent"
operation="starting elastic agent"
message="Enable elastic agent"
sub_name="Elastic Agent"

# Start Elastic Agent
Start_ElasticAgent()
{
  if [ "$(pidof systemd && echo "systemd" || echo "other")" = "other" ]; then
    log "INFO" "[Start_ElasticAgent] starting Elastic Agent"
    sudo service elastic-agent start
    log "INFO" "[Start_ElasticAgent] Elastic Agent started"
    write_status "$name" "$operation" "success" "$message" "$sub_name" "success" "Elastic Agent service has started" 2
  else
    log "INFO" "[Start_ElasticAgent] enabling and starting Elastic Agent"
    sudo systemctl enable elastic-agent
    sudo systemctl start elastic-agent
    log "INFO" "[Start_ElasticAgent] Elastic Agent started"
    write_status "$name" "$operation" "success" "$message" "$sub_name" "success" "Elastic Agent service has started" 2
  fi
}

retry_backoff Start_ElasticAgent
