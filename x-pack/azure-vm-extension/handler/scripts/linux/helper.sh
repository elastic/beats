#!/usr/bin/env bash
set -euo pipefail

DISTRO_OS=""
LOGS_FOLDER=""
CONFIG_FILE=""
CLOUD_ID=""
USERNAME=""
PASSWORD=""
ELASTICSEARCH_URL=""
STACK_VERSION=""
KIBANA_URL=""

checkOS()
{
  if dpkg -S /bin/ls >/dev/null 2>&1
then
  DISTRO_OS="DEB"
  log "[checkOS] distro is $DISTRO_OS" "INFO"
elif rpm -q -f /bin/ls >/dev/null 2>&1
then
  DISTRO_OS="RPM"
   log "[checkOS] distro is $DISTRO_OS" "INFO"
else
  DISTRO_OS="OTHER"
   log "[checkOS] distro is $DISTRO_OS" "INFO"
fi
}

install_dependencies() {
  checkOS

if [ "$DISTRO_OS" = "DEB" ]; then
  if [ $(dpkg-query -W -f='${Status}' curl 2>/dev/null | grep -c "ok installed") -eq 0 ]; then
  sudo apt-get --yes install  curl;
  fi
  if [ $(dpkg-query -W -f='${Status}' jq 2>/dev/null | grep -c "ok installed") -eq 0 ]; then
  sudo apt-get --yes install  jq;
  fi
elif [ "$DISTRO_OS" = "RPM" ]; then
   if ! rpm -qa | grep -qw jq; then
   yum install epel-release -y
   yum install jq -y
fi
else
  pacman -Qq | grep -qw jq || pacman -S jq
fi

}

get_logs_location()
{
  SCRIPT=$(readlink -f "$0")
  ES_EXT_DIR=$(dirname "$SCRIPT")
  ES_EXT_DIR="/mnt/c/work/beats/x-pack/azure-vm-extension/handler/"
   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]
then
    LOGS_FOLDER=$(jq -r '.[0].handlerEnvironment.logFolder' $ES_EXT_DIR/HandlerEnvironment.json)
else
    exit 1
fi
}

log()
{
  if [ "$LOGS_FOLDER" = "" ]; then
    get_logs_location
    fi

    echo \[$(date +%H:%M:%ST%d-%m-%Y)\]  "$1" "$2"
    echo \[$(date +%H:%M:%ST%d-%m-%Y)\]  "$1" "$2" >> $LOGS_FOLDER/es-agent-install.log
}



checkShasum ()
{
  local archive_file_name="${1}"
  local authentic_checksum_file="${2}"
  echo  --check <(grep "\s${archive_file_name}$" "${authentic_checksum_file}")

  if $(which sha256sum >/dev/null 2>&1); then
    sha256sum \
      --check <(grep "\s${archive_file_name}$" "${authentic_checksum_file}")
  elif $(which shasum >/dev/null 2>&1); then
    shasum \
      -a 256 \
      --check <(grep "\s${archive_file_name}$" "${authentic_checksum_file}")
  else
    echo "sha256sum or shasum is not available for use" >&2
    return 1
  fi
}


# configuration

get_configuration_location()
{

  SCRIPT=$(readlink -f "$0")
  ES_EXT_DIR=$(dirname "$SCRIPT")
  log "[get_configuration_location] main directory found $ES_EXT_DIR" "INFO"
  log "[get_configuration_location] looking for HandlerEnvironment.json file" "INFO"
  ES_EXT_DIR="/mnt/c/work/beats/x-pack/azure-vm-extension/handler/"

   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]
then
    log "[get_configuration_location] HandlerEnvironment.json file found" "INFO"
    config_folder=$(jq -r '.[0].handlerEnvironment.configFolder' $ES_EXT_DIR/HandlerEnvironment.json)
    log "[get_configuration_location]  configuration folder $config_folder found" "INFO"
    config_files_path="$config_folder/*.settings"
    CONFIG_FILE=$(ls $config_files_path 2>/dev/null | sort -V | tail -1)
    log "[get_configuration_location] configuration file $CONFIG_FILE found" "INFO"
else
    log "[get_configuration_location] HandlerEnvironment.json file not found" "ERROR"
    exit 1
fi
}


get_cloud_id()
{
get_configuration_location
log "INFO" "[get_cloud_id] Found configuration file $CONFIG_FILE"
if [ "$CONFIG_FILE" != "" ]; then
  CLOUD_ID=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.cloud_id' $CONFIG_FILE)
 log "INFO" "[get_cloud_id] Found cloud id $CLOUD_ID"
else
    log "[get_cloud_id] Configuration file not found" "ERROR"
    exit 1
  fi
}


get_username()
{
get_configuration_location
log "INFO" "[get_username] Found configuration file $CONFIG_FILE"
if [ "$CONFIG_FILE" != "" ]; then
 USERNAME=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.username' $CONFIG_FILE)
 log "INFO" "[get_cloud_id] Found username  $USERNAME"
 else
    log "[get_username] Configuration file not found" "ERROR"
    exit 1
  fi
}


get_password()
{
get_configuration_location
log "INFO" "[get_username] Found configuration file $CONFIG_FILE"
if [ "$CONFIG_FILE" != "" ]; then
 cert=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettingsCertThumbprint' $CONFIG_FILE)
 settings=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettings' $CONFIG_FILE)
 echo $settings
 echo $cert
  PASSWORD=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.password' $CONFIG_FILE)
  log "INFO" "[get_cloud_id] Found password  $PASSWORD"

 else
    log "[get_cloud_id] Configuration file not found" "ERROR"
    exit 1
  fi
}


get_kibana_host () {
  get_cloud_id
  if [ "$CLOUD_ID" != "" ]; then
 cloud_hash=$(echo $CLOUD_ID | cut -f2 -d:)
  cloud_tokens=$(echo $cloud_hash | base64 -d -)
  host_port=$(echo $cloud_tokens | cut -f1 -d$)
  KIBANA_URL="https://$(echo $cloud_tokens | cut -f3 -d$).${host_port}"
  log "INFO" "[es_agent_enroll] Found Kibana uri $KIBANA_URL"
 else
    log "ERROR" "[get_kibana_host] Cloud ID could not be parsed"
    exit 1
fi

}

get_elasticsearch_host () {
   get_cloud_id
  if [ "$CLOUD_ID" != "" ]; then
  cloud_hash=$(echo $CLOUD_ID | cut -f2 -d:)
  cloud_tokens=$(echo $cloud_hash | base64 -d -)
  host_port=$(echo $cloud_tokens | cut -f1 -d$)
  ELASTICSEARCH_URL="https://$(echo $cloud_tokens | cut -f2 -d$).${host_port}"
  log "[get_elasticsearch_host] Found ES uri $ELASTICSEARCH_URL" "INFO"
   else
    log "[get_elasticsearch_host] Cloud ID could not be parsed" "ERROR"
    exit 1
fi
}

get_cloud_stack_version () {
  log "INFO" "[get_cloud_stack_version] Get ES cluster URL"
  get_elasticsearch_host
  get_username
  get_password
   if [ "$ELASTICSEARCH_URL" != "" ] && [ "$USERNAME" != "" ] && [ "$PASSWORD" != "" ]; then
    jsonResult=$(curl "${ELASTICSEARCH_URL}"  -H 'Content-Type: application/json' -u ${USERNAME}:${PASSWORD})
      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "ERROR" "[get_cloud_stack_version] error pinging $ELASTICSEARCH_URL"
        exit $EXITCODE
      fi
      echo $jsonResult
   STACK_VERSION=$(echo $jsonResult | jq -r '.version.number')
   log "INFO" "[get_cloud_stack_version] Stack version found is $STACK_VERSION"
   else
    log "ERROR" "[get_cloud_stack_version] Elasticsearch URL could not be found"
    exit 1
fi
}






