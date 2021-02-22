#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

checkOS

install_dependencies
# Install Elastic Agent
install_es_agent_deb()
{
    local OS_SUFFIX="-amd64"
    local ALGORITHM="512"

    get_cloud_stack_version
    if [ $STACK_VERSION = "" ]; then
       log "ERROR" "[install_es_ag_deb] Stack version could not be found"
      exit 1
    else
    log "INFO" "[install_es_ag_deb] installing Elastic Agent $STACK_VERSION"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.deb"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_deb] download location - $DOWNLOAD_URL" "INFO"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
    log "ERROR" "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    log "INFO" "[install_es_ag_deb] downloaded Elastic Agent $STACK_VERSION"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_deb] error validating checksum for Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi

    sudo dpkg -i $PACKAGE
    log "INFO" "[install_es_ag_deb] installed Elastic Agent $STACK_VERSION"

 fi
}

install_es_agent_rpm()
{
    local OS_SUFFIX="-x86_64"
    local ALGORITHM="512"
    get_cloud_stack_version
    if [ $STACK_VERSION = "" ]; then
       log "ERROR" "[install_es_ag_deb] Stack version could not be found"
      exit 1
    else
      local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.rpm"
      local SHASUM="$PACKAGE.sha$ALGORITHM"
      local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
      local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"
      log "INFO" "[install_es_ag_rpm] installing Elastic Agent $STACK_VERSION"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum"
        exit $EXIT_CODE
    fi
    log "INFO" "[install_es_ag_rpm] download location - $DOWNLOAD_URL"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    log "INFO" "[install_es_ag_rpm] downloaded Elastic Agent $STACK_VERSION"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_rpm] error validating checksum for Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi

    sudo rpm -vi $PACKAGE
    log "INFO" "[install_es_ag_rpm] installed Elastic Agent $STACK_VERSION"
      fi

}

install_es_agent_linux()
{
    local OS_SUFFIX="-linux-x86_64"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.tar.gz"
    local ALGORITHM="512"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"


    log "INFO" "[install_es_ag_linux] installing Elastic Agent $STACK_VERSION"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_linux] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum"
        exit $EXIT_CODE
    fi
    log "INFO" "[install_es_ag_linux] download location - $DOWNLOAD_URL"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_linux] error downloading Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    log "INFO" "[install_es_ag_linux] downloaded Elastic Agent $STACK_VERSION"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "ERROR" "[install_es_ag_linux] error validating checksum for Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    tar xzvf $PACKAGE
    log "INFO" "[install_es_ag_linux] installed Elastic Agent $STACK_VERSION"
}



# Enroll Elastic Agent
enroll_es_agent() {
  get_kibana_host
  get_username
  get_password
   if [ "$KIBANA_URL" != "" ] && [ "$USERNAME" != "" ] && [ "$PASSWORD" != "" ]; then
  local ENROLLMENT_TOKEN_ID=""
  local ENROLLMENT_TOKEN=""
  jsonResult=$(curl "${KIBANA_URL}"/api/fleet/enrollment-api-keys  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} )
      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "ERROR" "[enroll_es_agent] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN"
        exit $EXITCODE
      fi
      ENROLLMENT_TOKEN_ID=$(echo $jsonResult | jq -r '.list[0].id')
       log "INFO" "[enroll_es_agent] ENROLLMENT_TOKEN_ID is $ENROLLMENT_TOKEN_ID"

      jsonResult=$(curl ${KIBANA_URL}/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN_ID \
        -H 'Content-Type: application/json' \
        -H 'kbn-xsrf: true' \
        -u ${USERNAME}:${PASSWORD} )
      EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "ERROR" "[enroll_es_agent] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN"
        exit $EXITCODE
      fi

      ENROLLMENT_TOKEN=$(echo $jsonResult | jq -r '.item.api_key')
      log "INFO" "[enroll_es_agent] ENROLLMENT_TOKEN is $ENROLLMENT_TOKEN"
      log "INFO" "[enroll_es_agent] Enrolling the Elastic Agent to Fleet ${KIBANA_URL}"
      elastic-agent enroll  "${KIBANA_URL}" "$ENROLLMENT_TOKEN" -f
else
   log "ERROR" "[enroll_es_agent] error retrieving user credentials"
   exit 1
  fi
}



if [ "$DISTRO_OS" = "DEB" ]; then
    install_es_agent_deb
elif [ "$DISTRO_OS" = "RPM" ]; then
    install_es_agent_rpm
else
  install_es_agent_linux
fi

log "INFO" "[es_agent_start] enrolling Elastic Agent $STACK_VERSION"
enroll_es_agent
log "INFO" "[es_agent_start] Elastic Agent $STACK_VERSION enrolled"
