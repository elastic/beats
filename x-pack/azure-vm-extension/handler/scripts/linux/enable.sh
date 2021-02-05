#!/usr/bin/env bash
set -euo pipefail
source ./helper.sh


# Enroll Elastic Agent
es_agent_enroll() {
  cloud_hash=$(echo $CLOUD_ID | cut -f2 -d:)
  cloud_tokens=$(echo $cloud_hash | base64 -d -)
  host_port=$(echo $cloud_tokens | cut -f1 -d$)
  local ELASTICSEARCH_URL="https://$(echo $cloud_tokens | cut -f2 -d$).${host_port}"
  local KIBANA_URL="https://$(echo $cloud_tokens | cut -f3 -d$).${host_port}"
  log "[es_agent_enroll] Found ES uri $ELASTICSEARCH_URL and Kibana host $KIBANA_URL" "INFO"

  jsonResult=$(curl "${KIBANA_URL}"/api/fleet/enrollment-api-keys  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} )

      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[es_agent_enroll] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi
      local ENROLLMENT_TOKEN=$(echo $jsonResult | jq -r '.list[0].id')
       log "[es_agent_enroll] ENROLLMENT_TOKEN is $ENROLLMENT_TOKEN" "INFO"

      jsonResult=$(curl ${KIBANA_URL}/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN \
        -H 'Content-Type: application/json' \
        -H 'kbn-xsrf: true' \
        -u ${USERNAME}:${PASSWORD} )

      EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[es_agent_enroll] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi

      ENROLLMENT_TOKEN=$(echo $jsonResult | jq -r '.item.api_key')
      log "[es_agent_enroll] ENROLLMENT_TOKEN is $ENROLLMENT_TOKEN" "INFO"
  log "[es_agent_enroll] Enrolling the Elastic Agent" "INFO"
  ./elastic-agent enroll  ${KIBANA_URL} $ENROLLMENT_TOKEN -f
}


# Start Elastic Agent
es_agent_start()
{
log "[es_agent_start] enrolling Elastic Agent $STACK_VERSION" "INFO"
es_agent_enroll
log "[es_agent_start] Elastic Agent $STACK_VERSION enrolled" "INFO"

log "[es_agent_start] enabling Elastic Agent $STACK_VERSION" "INFO"
sudo systemctl enable elastic-agent
log "[es_agent_start] Elastic Agent $STACK_VERSION enabled" "INFO"

log "[es_agent_start] starting Elastic Agent $STACK_VERSION" "INFO"
sudo systemctl start elastic-agent
log "[es_agent_start] Elastic Agent $STACK_VERSION started" "INFO"

#sudo service elastic-agent start
}

es_agent_start
