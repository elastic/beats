#!/usr/bin/env bash
set -euo pipefail

DISTRO_OS=""
LOGS_FOLDER=""
CONFIG_FILE=""
STATUS_FOLDER=""
CLOUD_ID=""
USERNAME=""
PASSWORD=""
ELASTICSEARCH_URL=""
STACK_VERSION=""
KIBANA_URL=""
POLICY_ID=""

checkOS()
{
  if dpkg -S /bin/ls >/dev/null 2>&1
  then
    DISTRO_OS="DEB"
    echo "[checkOS] distro is $DISTRO_OS" "INFO"
  elif rpm -q -f /bin/ls >/dev/null 2>&1
  then
    DISTRO_OS="RPM"
    echo "[checkOS] distro is $DISTRO_OS" "INFO"
  else
    DISTRO_OS="OTHER"
    echo "[checkOS] distro is $DISTRO_OS" "INFO"
  fi
}

install_dependencies() {
  checkOS
  if [ "$DISTRO_OS" = "DEB" ]; then
    sudo apt-get update
  if [ $(dpkg-query -W -f='${Status}' curl 2>/dev/null | grep -c "ok installed") -eq 0 ]; then
    #sudo apt-get --yes install  curl;
    (sudo apt-get --yes install  curl || (sleep 15; sudo apt-get --yes install  curl))
  fi
  if [ $(dpkg-query -W -f='${Status}' jq 2>/dev/null | grep -c "ok installed") -eq 0 ]; then
    #sudo apt-get --yes install  jq;
    (sudo apt-get --yes install  jq || (sleep 15; apt-get --yes install  jq))
  fi
  elif [ "$DISTRO_OS" = "RPM" ]; then
    #sudo yum update -y --disablerepo='*' --enablerepo='*microsoft*'
    if ! rpm -qa | grep -qw jq; then
      #yum install epel-release -y
      yum install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm -y
      yum install jq -y
    fi
  else
    pacman -Qq | grep -qw jq || pacman -S jq
  fi
}

get_logs_location()
{
  SCRIPT=$(readlink -f "$0")
  cmd_path=$(dirname "$SCRIPT")
  ES_EXT_DIR=$(cd "$( dirname "${cmd_path}" )" >/dev/null 2>&1 && pwd)
   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]; then
    LOGS_FOLDER=$(jq -r '.[0].handlerEnvironment.logFolder' $ES_EXT_DIR/HandlerEnvironment.json)
  else
    exit 1
  fi
}

get_status_location()
{
  SCRIPT=$(readlink -f "$0")
  cmd_path=$(dirname "$SCRIPT")
  ES_EXT_DIR=$(cd "$( dirname "${cmd_path}" )" >/dev/null 2>&1 && pwd)
   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]
then
    STATUS_FOLDER=$(jq -r '.[0].handlerEnvironment.statusFolder' $ES_EXT_DIR/HandlerEnvironment.json)
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
    echo \[$(date +%H:%M:%ST%d-%m-%Y)\]  "$1" "$2" >> $LOGS_FOLDER/es-agent.log
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

write_status() {
  get_status_location
  if [[ "$STATUS_FOLDER" != "" ]]; then
    status_files_path="$STATUS_FOLDER/*.status"
    latest_status_file=$(ls $status_files_path 2>/dev/null | sort -V | tail -1)
    if [[ $latest_status_file = "" ]]; then
      echo ""
      fi
    fi
}


# configuration

get_configuration_location()
{
  SCRIPT=$(readlink -f "$0")
  cmd_path=$(dirname "$SCRIPT")
  ES_EXT_DIR=$(cd "$( dirname "${cmd_path}" )" >/dev/null 2>&1 && pwd)
  log "INFO" "[get_configuration_location] main directory found $ES_EXT_DIR"
  log "INFO" "[get_configuration_location] looking for HandlerEnvironment.json file"
  if [ -e "$ES_EXT_DIR/HandlerEnvironment.json" ]; then
    log "INFO" "[get_configuration_location] HandlerEnvironment.json file found"
    config_folder=$(jq -r '.[0].handlerEnvironment.configFolder' "$ES_EXT_DIR/HandlerEnvironment.json")
    log "INFO" "[get_configuration_location]  configuration folder $config_folder found"
    config_files_path="$config_folder/*.settings"
    CONFIG_FILE=$(ls $config_files_path 2>/dev/null | sort -V | tail -1)
    log "INFO" "[get_configuration_location] configuration file $CONFIG_FILE found"
  else
    log "ERROR" "[get_configuration_location] HandlerEnvironment.json file not found"
    exit 1
  fi
}


get_cloud_id()
{
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    log "INFO" "[get_cloud_id] Found configuration file $CONFIG_FILE"
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
    log "INFO" "[get_username] Found username  $USERNAME"
  else
    log "ERROR" "[get_username] Configuration file not found"
    exit 1
  fi
}


get_password()
{
  get_configuration_location
  log "INFO" "[get_password] Found configuration file $CONFIG_FILE"
  if [ "$CONFIG_FILE" != "" ]; then
    cert=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettingsCertThumbprint' $CONFIG_FILE)
    settings=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettings' $CONFIG_FILE)
    PASSWORD=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.password' $CONFIG_FILE)
    log "INFO" "[get_password] Found password  $PASSWORD"
  else
    log "ERROR" "[get_password] Configuration file not found"
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
    log "INFO" "[get_kibana_host] Found Kibana uri $KIBANA_URL"
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
    log "INFO" "[get_elasticsearch_host] Found ES uri $ELASTICSEARCH_URL"
  else
    log "ERROR" "[get_elasticsearch_host] Cloud ID could not be parsed"
    exit 1
  fi
}

get_cloud_stack_version () {
  log "INFO" "[get_cloud_stack_version] Get ES cluster URL"
  get_elasticsearch_host
  get_username
  get_password
  if [ "$ELASTICSEARCH_URL" != "" ] && [ "$USERNAME" != "" ] && [ "$PASSWORD" != "" ]; then
    json_result=$(curl "${ELASTICSEARCH_URL}"  -H 'Content-Type: application/json' -u ${USERNAME}:${PASSWORD})
    local EXITCODE=$?
    if [ $EXITCODE -ne 0 ]; then
      log "ERROR" "[get_cloud_stack_version] error pinging $ELASTICSEARCH_URL"
      exit $EXITCODE
    fi
   STACK_VERSION=$(echo $json_result | jq -r '.version.number')
   log "INFO" "[get_cloud_stack_version] Stack version found is $STACK_VERSION"
  else
    log "ERROR" "[get_cloud_stack_version] Elasticsearch URL could not be found"
    exit 1
  fi
}

function parse_yaml {
   local s='[[:space:]]*' w='[a-zA-Z0-9_]*' fs=$(echo @|tr @ '\034')
   sed -ne "s|^\($s\):|\1|" \
        -e "s|^\($s\)\($w\)$s:$s[\"']\(.*\)[\"']$s\$|\1$fs\2$fs\3|p" \
        -e "s|^\($s\)\($w\)$s:$s\(.*\)$s\$|\1$fs\2$fs\3|p"  $1 |
   awk -F$fs '{
      indent = length($1)/2;
      vname[indent] = $2;
      for (i in vname) {if (i > indent) {delete vname[i]}}
      if (length($3) > 0) {
         vn=""; for (i=0; i<indent; i++) {vn=(vn)(vname[i])("_")}
         printf("%s%s=\"%s\"\n",vn, $2, $3);
      }
   }'
}

function retry_backoff() {
  local attempts=3
  local sleep_millis=20000
  # shift 3
  for attempt in `seq 1 $attempts`; do
    if [[ $attempt -gt 1 ]]; then
      log "ERROR" "[retry_backoff] Function failed on attempt $attempt, retrying in 20 sec ..."
    fi
    "$@" && local rc=$? || local rc=$?
    if [[ ! $rc -gt 0 ]]; then
      return $rc
    fi
    if [[ $attempt -eq $attempts ]]; then
      log "ERROR" "[retry_backoff] Function failed on last attempt $attempt."
      exit 1
    fi
    local sleep_ms="$(($sleep_millis))"
    sleep "${sleep_ms:0:-3}.${sleep_ms: -3}"
  done
}

get_default_policy() {
   eval result="$1"
   list=$(echo "$result" | jq -r '.list')
   for row in $(echo "${list}" | jq -r '.[] | @base64'); do
   _jq() {
     echo ${row} | base64 --decode | jq -r ${1}
    }
  name=$(_jq '.name')
  is_active=$(_jq '.active')
  if [[ "$name" == *"Default"* ]]  && [[ "$is_active" = "true" ]]; then
  POLICY_ID=$(_jq '.id')
  fi
done
}

get_any_active_policy() {
   eval result="$1"
   list=$(echo "$result" | jq -r '.list')
   for row in $(echo "${list}" | jq -r '.[] | @base64'); do
   _jq() {
     echo ${row} | base64 --decode | jq -r ${1}
    }
  is_active=$(_jq '.active')
  if [[ "$is_active" = "true" ]]; then
  POLICY_ID=$(_jq '.id')
  fi
done
}
