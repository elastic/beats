#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

checkOS

unenroll_es_agent_deb_rpm()
{
  get_kibana_host
  get_username
  get_password
   if [ "$KIBANA_URL" != "" ] && [ "$USERNAME" != "" ] && [ "$PASSWORD" != "" ]; then
     echo "here $KIBANA_URL"
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

     #jsonResult=$(curl -X DELETE "${KIBANA_URL}"/api/fleet/agents/ubuntunew/unenroll/  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} )
      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[unenroll_es_agent] error calling $KIBANA_URL/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN_ID in order to deactivate the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi
      ENROLLMENT_TOKEN_ID=$(echo $jsonResult | jq -r '.list[0].id')
       log "[unenroll_es_agent] ENROLLMENT_TOKEN_ID is $ENROLLMENT_TOKEN_ID" "INFO"
     fi

}


uninstall_es_agent()
{
  log "INFO" "[uninstall_es_agent] Stopping Elastic Agent"
  sudo service elastic-agent stop
  log "INFO" "[uninstall_es_agent] Elastic Agent is stopped"
  log "INFO" "[uninstall_es_agent] Unenrolling Elastic Agent"
  unenroll_es_agent_deb_rpm
  log "INFO" "[uninstall_es_agent] Elastic Agent has been unenrolled"
  if [ "$(pidof systemd && echo "systemd" || echo "other")" = "other" ]; then
    log "INFO" "[uninstall_es_agent] removing Elastic Agent directories"
    sudo service elastic-agent start
    log "INFO" "[uninstall_es_agent] Elastic Agent removed"
  else
    log "INFO" "[uninstall_es_agent] removing Elastic Agent directories"

    sudo rm -rf /usr/share/elastic-agent
    sudo rm -rf /etc/elastic-agent
    sudo rm -rf /var/lib/elastic-agent
    sudo rm -rf /usr/bin/elastic-agent
    systemctl daemon-reload
    systemctl reset-failed
    log "INFO" "[uninstall_es_agent] Elastic Agent removed"
  fi



}




log "Uninstalling Elastic Agent" "INFO"
if [ "$DISTRO_OS" = "DEB" ] || [ "$DISTRO_OS" = "RPM" ]; then
    uninstall_es_agent
else
  elastic-agent uninstall
fi

log "Elastic Agent is uninstalled" "INFO"
