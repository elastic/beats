#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# Start Elastic Agent
es_agent_start()
{
log "[es_agent_start] enabling Elastic Agent $STACK_VERSION" "INFO"
sudo systemctl enable elastic-agent
log "[es_agent_start] Elastic Agent $STACK_VERSION enabled" "INFO"

log "[es_agent_start] starting Elastic Agent $STACK_VERSION" "INFO"
sudo service elastic-agent start
log "[es_agent_start] Elastic Agent $STACK_VERSION started" "INFO"

}

es_agent_start
