#!/usr/bin/env bash
set -euo pipefail

DISTRO_OS=""
LOGS_FOLDER=""
CONFIG_FILE=""
STATUS_FOLDER=""
CLOUD_ID=""
USERNAME=""
PASSWORD=""
BASE64_AUTH=""
ELASTICSEARCH_URL=""
STACK_VERSION=""
KIBANA_URL=""
POLICY_ID=""
LINUX_CERT_PATH="/var/lib/waagent"
IS_NEW_CONFIG=""
OLD_KIBANA_URL=""
OLD_USERNAME=""
OLD_PASSWORD=""
OLD_BASE64_AUTH=""
OLD_CONFIG_FILE=""
OLD_CLOUD_ID=""
OLD_PROTECTED_SETTINGS=""
OLD_THUMBPRINT=""

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
  ES_EXT_DIR=$(dirname "$SCRIPT")
   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]; then
    LOGS_FOLDER=$(jq -r '.[0].handlerEnvironment.logFolder' $ES_EXT_DIR/HandlerEnvironment.json)
  else
    exit 1
  fi
}

get_status_location()
{
  SCRIPT=$(readlink -f "$0")
  ES_EXT_DIR=$(dirname "$SCRIPT")
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
  ES_EXT_DIR=$(dirname "$SCRIPT")
  if [ -e "$ES_EXT_DIR/HandlerEnvironment.json" ]; then
    log "INFO" "[get_configuration_location] HandlerEnvironment.json file found"
    config_folder=$(jq -r '.[0].handlerEnvironment.configFolder' "$ES_EXT_DIR/HandlerEnvironment.json")
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
    CLOUD_ID=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.cloudId' $CONFIG_FILE)
    log "INFO" "[get_cloud_id] Found cloud id $CLOUD_ID"
  else
    log "[get_cloud_id] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_protected_settings()
{
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    PROTECTED_SETTINGS=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettings' $CONFIG_FILE)
    log "INFO" "[get_protected_settings] Found protected settings"
  else
    log "[get_protected_settings] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_thumbprint()
{
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    THUMBPRINT=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettingsCertThumbprint' $CONFIG_FILE)
    log "INFO" "[get_thumbprint] Found thumbprint $THUMBPRINT"
  else
    log "[get_thumbprint] Configuration file not found" "ERROR"
    exit 1
  fi
}


get_username()
{
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    USERNAME=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.username' $CONFIG_FILE)
    log "INFO" "[get_username] Found username  $USERNAME"
  else
    log "ERROR" "[get_username] Configuration file not found"
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
  if [ "$ELASTICSEARCH_URL" = "" ]; then
    log "ERROR" "[get_cloud_stack_version] Elasticsearch URL could not be found"
    exit 1
  fi
  get_password
  get_base64Auth
   if [ "$PASSWORD" = "" ] && [ "$BASE64_AUTH" = "" ]; then
    log "ERROR" "[get_cloud_stack_version] Both PASSWORD and BASE64AUTH key could not be found"
    exit 1
  fi
  local cred=""
  if [ "$PASSWORD" != "" ] && [ "$PASSWORD" != "null" ]; then
    get_username
    if [ "$USERNAME" = "" ]; then
      log "ERROR" "[get_cloud_stack_version] USERNAME could not be found"
      exit 1
    fi
    cred=${USERNAME}:${PASSWORD}
  else
    cred=$(echo "$BASE64_AUTH" | base64 --decode)
  fi
  json_result=$(curl "${ELASTICSEARCH_URL}"  -H 'Content-Type: application/json' -u $cred)
  local EXITCODE=$?
  if [ $EXITCODE -ne 0 ]; then
      log "ERROR" "[get_cloud_stack_version] error pinging $ELASTICSEARCH_URL"
      exit $EXITCODE
  fi
  STACK_VERSION=$(echo $json_result | jq -r '.version.number')
  log "INFO" "[get_cloud_stack_version] Stack version found is $STACK_VERSION"
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

write_status() {
  local name="${1}"
  local operation="${2}"
  local mainStatus="${3}"
  local message="${4}"
  local subName="${5}"
  local subStatus="${6}"
  local subMessage="${7}"
  local sequenceNumber="0"
  local code=0
  get_status_location
  #2013-11-17T16:05:14Z
  timestampUTC=$(date +"%Y-%m-%dT%H:%M:%S%z")
  if [[ $subStatus = "error" ]]; then
        code=1
  fi
  if [[ "$STATUS_FOLDER" != "" ]]; then
    get_configuration_location
    if [ "$CONFIG_FILE" != "" ]; then
      filename="$(basename -- $CONFIG_FILE)"
      sequenceNumber=$(echo $filename | cut -f1 -d.)
    else
    log "[write_status] Configuration file not found" "ERROR"
    exit 1
    fi
  json="[{\"version\":\"1.0\",\"timestampUTC\":\"$timestampUTC\",\"status\":{\"name\":\"$name\",\"operation\":\"$operation\",\"status\":\"$mainStatus\",\"formattedMessage\": { \"lang\":\"en-US\", \"message\":\"$message\"},\"substatus\": [{ \"name\":\"$subName\", \"status\":\"$subStatus\",\"code\":\"$code\",\"formattedMessage\": { \"lang\":\"en-US\", \"message\":\"$subMessage\"}}]}} ]"
  echo $json > "$STATUS_FOLDER"/"$sequenceNumber".status
  fi
}

service_exists() {
    local n=$1
    if [[ $(systemctl list-units --all -t service --full --no-legend "$n.service" | cut -f1 -d' ') == $n.service ]]; then
        return 0
    else
        return 1
    fi
}

# encryption

encrypt() {
  cert_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.crt"
  private_key_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    openssl cms -encrypt -in <(echo "$2") -inkey $private_key_path -recip $cert_path -inform dem
  else
    echo "ERROR" "[decrypt] Decryption failed. Could not find certificates"
  exit 1
  fi
}

get_password() {
  get_protected_settings
  get_thumbprint
  cert_path="$LINUX_CERT_PATH/$THUMBPRINT.crt"
  private_key_path="$LINUX_CERT_PATH/$THUMBPRINT.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    protected_settings=$(openssl cms -decrypt -in <(echo "$PROTECTED_SETTINGS" | base64 --decode) -inkey "$private_key_path" -recip "$cert_path" -inform dem)
    PASSWORD=$(echo "$protected_settings" | jq -r '.password')
  else
    log "ERROR" "[get_password] Decryption failed. Could not find certificates"
    exit 1
  fi
}

get_base64Auth() {
  get_protected_settings
  get_thumbprint
  cert_path="$LINUX_CERT_PATH/$THUMBPRINT.crt"
  private_key_path="$LINUX_CERT_PATH/$THUMBPRINT.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    protected_settings=$(openssl cms -decrypt -in <(echo "$PROTECTED_SETTINGS" | base64 --decode) -inkey "$private_key_path" -recip "$cert_path" -inform dem)
    BASE64_AUTH=$(echo "${protected_settings}" | jq -r '.base64Auth')
  else
    log "ERROR" "[get_base64Auth] Decryption failed. Could not find certificates"
    exit 1
  fi
}

# update config

is_new_config(){
  currentSequence=""
  newSequence=""
  isUpdate=""
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    filename="$(basename -- $CONFIG_FILE)"
    newSequence=$(echo $filename | cut -f1 -d.)
  else
    log "[get_sequence] Configuration file not found" "ERROR"
    exit 1
  fi
  if [ "$LOGS_FOLDER" = "" ]; then
      get_logs_location
  fi
  if [ -f "$LOGS_FOLDER/update.txt" ]; then
    isUpdate=true
  else
    isUpdate=false
  fi
  if [ -f "$LOGS_FOLDER/current.sequence" ]; then
    currentSequence=$(< "$LOGS_FOLDER/current.sequence")
  else
    currentSequence=""
  fi
  log "INFO" "[is_new_config] Current sequence is $currentSequence and new sequence is $newSequence"
  if [[ "$newSequence" = "" ]]; then
    IS_NEW_CONFIG=false
  elif   [[ "$isUpdate" = true ]]; then
    log "INFO" "[is_new_config] Part of the update"
    IS_NEW_CONFIG=false
  elif   [[ "$newSequence" = "$currentSequence" ]]; then
    IS_NEW_CONFIG=false
  else
      IS_NEW_CONFIG=true
  fi
}
set_update_var() {
  log "INFO" "[set_update_var] Verified update"
  if [ "$LOGS_FOLDER" = "" ]; then
      get_logs_location
  fi
  echo "1" > "$LOGS_FOLDER/update.txt"
}

function set_sequence_to_file
{
  log "INFO" "[set_sequence_to_file] Setting new sequence"
  get_configuration_location
  if [ "$CONFIG_FILE" != "" ]; then
    filename="$(basename -- $CONFIG_FILE)"
    newSequence=$(echo $filename | cut -f1 -d.)
    if [ "$LOGS_FOLDER" = "" ]; then
      get_logs_location
    fi
    #json="{\"sequence\":\"$newSequence\",\"update\":\"false\"}"
    echo "$newSequence" > "$LOGS_FOLDER/current.sequence"
    rm "$LOGS_FOLDER/update.txt"
    log "INFO" "[set_sequence_to_file] Sequence has been set"
  else
    log "[set_sequence_to_file] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_prev_configuration_location()
{
  SCRIPT=$(readlink -f "$0")
  ES_EXT_DIR=$(dirname "$SCRIPT")
  log "INFO" "[get_prev_configuration_location] main directory found $ES_EXT_DIR"
  if [ -e "$ES_EXT_DIR/HandlerEnvironment.json" ]; then
    log "INFO" "[get_prev_configuration_location] HandlerEnvironment.json file found"
    config_folder=$(jq -r '.[0].handlerEnvironment.configFolder' "$ES_EXT_DIR/HandlerEnvironment.json")
    log "INFO" "[get_prev_configuration_location]  configuration folder $config_folder found"
    config_files_path="$config_folder/*.settings"
    OLD_CONFIG_FILE=$(ls $config_files_path 2>/dev/null | sort -V | tail -n 2 | head -n 1)
    log "INFO" "[get_prev_configuration_location] configuration file $OLD_CONFIG_FILE found"
  else
    log "ERROR" "[get_prev_configuration_location] HandlerEnvironment.json file not found"
    exit 1
  fi
}

get_prev_username()
{
  get_prev_configuration_location
  if [ "$OLD_CONFIG_FILE" != "" ]; then
    OLD_USERNAME=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.username' $OLD_CONFIG_FILE)
    log "INFO" "[get_prev_username] Found username  OLD_USERNAME"
  else
    log "ERROR" "[get_prev_username] Configuration file not found"
    exit 1
  fi
}

get_prev_cloud_id()
{
  get_prev_configuration_location
  if [ "$OLD_CONFIG_FILE" != "" ]; then
    OLD_CLOUD_ID=$(jq -r '.runtimeSettings[0].handlerSettings.publicSettings.cloudId' $OLD_CONFIG_FILE)
    log "INFO" "[get_prev_cloud_id] Found cloud id $OLD_CLOUD_ID"
  else
    log "[get_prev_cloud_id] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_prev_kibana_host () {
  get_prev_cloud_id
  if [ "$OLD_CLOUD_ID" != "" ]; then
    cloud_hash=$(echo $OLD_CLOUD_ID | cut -f2 -d:)
    cloud_tokens=$(echo $cloud_hash | base64 -d -)
    host_port=$(echo $cloud_tokens | cut -f1 -d$)
    OLD_KIBANA_URL="https://$(echo $cloud_tokens | cut -f3 -d$).${host_port}"
    log "INFO" "[get_prev_kibana_host] Found Kibana uri $OLD_KIBANA_URL"
 else
    log "ERROR" "[get_prev_kibana_host] Cloud ID could not be parsed"
    exit 1
  fi

}

get_prev_protected_settings()
{
  get_prev_configuration_location
  if [ "$OLD_CONFIG_FILE" != "" ]; then
    OLD_PROTECTED_SETTINGS=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettings' $OLD_CONFIG_FILE)
    log "INFO" "[get_prev_protected_settings] Found protected settings $OLD_PROTECTED_SETTINGS"
  else
    log "[get_prev_protected_settings] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_prev_thumbprint()
{
  get_prev_configuration_location
  if [ "$OLD_CONFIG_FILE" != "" ]; then
    OLD_THUMBPRINT=$(jq -r '.runtimeSettings[0].handlerSettings.protectedSettingsCertThumbprint' $OLD_CONFIG_FILE)
    log "INFO" "[get_prev_thumbprint] Found thumbprint $OLD_THUMBPRINT"
  else
    log "[get_prev_thumbprint] Configuration file not found" "ERROR"
    exit 1
  fi
}

get_prev_password() {
  get_prev_protected_settings
  get_prev_thumbprint
  cert_path="$LINUX_CERT_PATH/$OLD_THUMBPRINT.crt"
  private_key_path="$LINUX_CERT_PATH/$OLD_THUMBPRINT.prv"
  log "INFO" "Found cerficate $cert_path and $private_key_path"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    protected_settings=$(openssl cms -decrypt -in <(echo "$OLD_PROTECTED_SETTINGS" | base64 --decode) -inkey "$private_key_path" -recip "$cert_path" -inform dem)
    OLD_PASSWORD=$(echo "$protected_settings" | jq -r '.password')
  else
    log "ERROR" "[get_prev_password] Decryption failed. Could not find certificates"
    exit 1
  fi
}

get_prev_base64Auth() {
  get_prev_protected_settings
  get_prev_thumbprint
  cert_path="$LINUX_CERT_PATH/$OLD_THUMBPRINT.crt"
  private_key_path="$LINUX_CERT_PATH/$OLD_THUMBPRINT.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    protected_settings=$(openssl cms -decrypt -in <(echo "$OLD_PROTECTED_SETTINGS" | base64 --decode) -inkey "$private_key_path" -recip "$cert_path" -inform dem)
    OLD_BASE64_AUTH=$(echo "${protected_settings}" | jq -r '.base64Auth')
  else
    log "ERROR" "[get_prev_base64Auth] Decryption failed. Could not find certificates"
    exit 1
  fi
}
