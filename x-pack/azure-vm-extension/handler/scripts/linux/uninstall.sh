#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

checkOS

unenroll_es_agent_deb_rpm()
{
  log "INFO" "[unenroll_es_agent] Unenrolling elastic agent"
  get_kibana_host
  get_username
  get_password
   if [ "$KIBANA_URL" != "" ] && [ "$USERNAME" != "" ] && [ "$PASSWORD" != "" ]; then
     echo "here $KIBANA_URL"
     eval $(parse_yaml "/etc/elastic-agent/fleet.yml")
     log "INFO" "[unenroll_es_agent] Agent ID is $agent_id"
     jsonResult=$(curl -X POST "${KIBANA_URL}/api/fleet/agents/$agent_id/unenroll"  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} --data '{"force":"true"}' )
     echo $jsonResult
     local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[unenroll_es_agent] error calling $KIBANA_URL/api/fleet/agents/$agent_id/unenroll in order to unenroll the agent" "ERROR"
        exit $EXITCODE
      fi
       log "INFO" "[unenroll_es_agent] Agent has been unenrolled"
     fi

}


uninstall_es_agent()
{
  log "INFO" "[uninstall_es_agent] Unenrolling Elastic Agent"
  unenroll_es_agent_deb_rpm
  log "INFO" "[uninstall_es_agent] Elastic Agent has been unenrolled"
  if [ "$(pidof systemd && echo "systemd" || echo "other")" == "other" ]; then
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
