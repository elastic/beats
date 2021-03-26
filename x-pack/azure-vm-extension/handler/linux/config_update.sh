#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

# for status
name="Uninstall elastic agent"
first_operation="unenrolling elastic agent"
second_operation="uninstalling elastic agent and removing any elastic agent related folders"
message="Uninstall elastic agent"
sub_name="Elastic Agent"

checkOS

Unenroll_Old_ElasticAgent_DEB_RPM()
{
  log "INFO" "[Unenroll_Old_ElasticAgent_DEB_RPM] Unenrolling elastic agent"
  get_prev_kibana_host
  if [[ "$OLD_KIBANA_URL" = "" ]]; then
    log "ERROR" "[Unenroll_Old_ElasticAgent_DEB_RPM] Kibana URL could not be found/parsed"
    return 1
  fi
  get_prev_password
  get_prev_base64Auth
  if [ "$OLD_PASSWORD" = "" ] && [ "$OLD_BASE64_AUTH" = "" ]; then
    log "ERROR" "[Unenroll_Old_ElasticAgent_DEB_RPM] Password could not be found/parsed"
    return 1
  fi
  local cred=""
  if [[ "$OLD_PASSWORD" != "" ]] && [ "$OLD_PASSWORD" != "null" ]; then
    get_prev_username
    if [[ "$OLD_USERNAME" = "" ]]; then
      log "ERROR" "[Unenroll_Old_ElasticAgent_DEB_RPM] Username could not be found/parsed"
      return 1
    fi
    cred=${OLD_USERNAME}:${OLD_PASSWORD}
  else
    cred=$(echo "$OLD_BASE64_AUTH" | base64 --decode)
  fi
  eval $(parse_yaml "/etc/elastic-agent/fleet.yml")
  if [[ "$agent_id" = "" ]]; then
    log "ERROR" "[Unenroll_Old_ElasticAgent_DEB_RPM] Password could not be found/parsed"
    return 1
  fi
  log "INFO" "[Unenroll_Old_ElasticAgent_DEB_RPM] Agent ID is $agent_id"
  jsonResult=$(curl -X POST "${OLD_KIBANA_URL}/api/fleet/agents/$agent_id/unenroll"  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u "$cred" --data '{"force":"true"}' )
  local EXITCODE=$?
  if [ $EXITCODE -ne 0 ]; then
    log "ERROR" "[Unenroll_Old_ElasticAgent_DEB_RPM] error calling $OLD_KIBANA_URL/api/fleet/agents/$agent_id/unenroll in order to unenroll the agent"
    return $EXITCODE
  fi
  log "INFO" "[Unenroll_Old_ElasticAgent_DEB_RPM] Agent has been unenrolled"
  write_status "$name" "$first_operation" "success" "$message" "$sub_name" "success" "Elastic Agent service has been unenrolled"
}

Uninstall_Old_ElasticAgent_DEB_RPM() {
   if [ "$DISTRO_OS" = "RPM" ]; then
      sudo rpm -e elastic-agent
   fi
   log "INFO" "[Uninstall_Old_ElasticAgent_DEB_RPM] removing Elastic Agent directories"
   sudo systemctl stop elastic-agent
   sudo systemctl disable elastic-agent
   sudo rm -rf /usr/share/elastic-agent
   sudo rm -rf /etc/elastic-agent
   sudo rm -rf /var/lib/elastic-agent
   sudo rm -rf /usr/bin/elastic-agent
   sudo systemctl daemon-reload
   sudo systemctl reset-failed
   if [ "$DISTRO_OS" = "DEB" ]; then
     sudo dpkg -r elastic-agent
     sudo dpkg -P elastic-agent
   fi
   log "INFO" "[Uninstall_Old_ElasticAgent_DEB_RPM] Elastic Agent removed"
}


Uninstall_Old_ElasticAgent()
{
  log "INFO" "[Uninstall_Old_ElasticAgent] Unenrolling Elastic Agent"
  retry_backoff Unenroll_Old_ElasticAgent_DEB_RPM
  log "INFO" "[Uninstall_Old_ElasticAgent] Elastic Agent has been unenrolled"
  if [ "$(pidof systemd && echo "systemd" || echo "other")" == "other" ]; then
    sudo elastic-agent uninstall
    log "INFO" "[Uninstall_Old_ElasticAgent] Elastic Agent removed"
  else
    retry_backoff Uninstall_Old_ElasticAgent_DEB_RPM
  fi
  log "INFO" "Elastic Agent is uninstalled"
  write_status "$name" "$second_operation" "success" "$message" "$sub_name" "error" "Elastic Agent service has been uninstalled"
}


