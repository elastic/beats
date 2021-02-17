#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh




unenroll_es_agent_deb_rpm()
{
  KIBANA_URL=$( get_kibana_host )
  local ENROLLMENT_TOKEN_ID=""
  jsonResult=$(curl "${KIBANA_URL}"/api/fleet/enrollment-api-keys  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} )

      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[unenroll_es_agent] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi
      ENROLLMENT_TOKEN_ID=$(echo $jsonResult | jq -r '.list[0].id')
      log "[unenroll_es_agent] ENROLLMENT_TOKEN_ID is $ENROLLMENT_TOKEN_ID" "INFO"

     jsonResult=$(curl -X DELETE "${KIBANA_URL}"/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN_ID  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} -d {"action":"deleted"})

      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[unenroll_es_agent] error calling $KIBANA_URL/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN_ID in order to deactivate the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi
      ENROLLMENT_TOKEN_ID=$(echo $jsonResult | jq -r '.list[0].id')
       log "[unenroll_es_agent] ENROLLMENT_TOKEN_ID is $ENROLLMENT_TOKEN_ID" "INFO"
}


uninstall_es_agent()
{
  log "Unenrolling Elastic Agent" "INFO"
  unenroll_es_agent
  log "Elastic Agent has been unenrolled" "INFO"
  log "Stopping Elastic Agent" "INFO"
  sudo service elastic-agent stop
  log "Elastic Agent is stopped" "INFO"

}




log "Uninstalling Elastic Agent" "INFO"

if [ "$DISTRO_OS" = "DEB" ]; then
    uninstall_es_ag
elif [ "$DISTRO_OS" = "RPM" ]; then
    uninstall_es_ag
else
  elastic-agent uninstall
fi

log "Elastic Agent is uninstalled" "INFO"
