#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# Start Elastic Agent
start_es_agent()
{
log "INFO" "[es_agent_start] enabling Elastic Agent $STACK_VERSION"
sudo systemctl enable elastic-agent
log "INFO" "[es_agent_start] Elastic Agent $STACK_VERSION enabled"

log "INFO" "[es_agent_start] starting Elastic Agent $STACK_VERSION"
sudo service elastic-agent start
log "INFO" "[es_agent_start] Elastic Agent $STACK_VERSION started"

}

start_es_agent
