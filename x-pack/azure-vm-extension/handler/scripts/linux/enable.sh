#!/usr/bin/env bash
set -euo pipefail

log()
{
    echo \[$(date +%d%m%Y-%H:%M:%S)\] "$1"
    echo \[$(date +%d%m%Y-%H:%M:%S)\] "$1" >> /var/log/es-agent-install.log
}

# Start Elastic Agent
start_es_ag()
{
log "[start_es_ag] enrolling Elastic Agent $STACK_VERSION"
sudo elastic-agent enroll $KIBANA_URL $ENROLLMENT_TOKEN
log "[start_es_ag] Elastic Agent $STACK_VERSION enrolled"

log "[start_es_ag] enabling Elastic Agent $STACK_VERSION"
sudo systemctl enable elastic-agent
log "[start_es_ag] Elastic Agent $STACK_VERSION enabled"

log "[start_es_ag] starting Elastic Agent $STACK_VERSION"
sudo systemctl start elastic-agent
log "[start_es_ag] Elastic Agent $STACK_VERSION started"

#sudo service elastic-agent start
}

start_es_ag
