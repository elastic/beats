#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

install_dependencies
# Install Elastic Agent
install_es_ag_deb()
{
    local OS_SUFFIX="-amd64"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.deb"
    local ALGORITHM="512"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"

    log "[install_es_ag_deb] installing Elastic Agent $STACK_VERSION" "INFO"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_deb] download location - $DOWNLOAD_URL" "INFO"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
    log "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_deb] downloaded Elastic Agent $STACK_VERSION" "INFO"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_deb] error validating checksum for Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi

    sudo dpkg -i $PACKAGE
    log "[install_es_ag_deb] installed Elastic Agent $STACK_VERSION" "INFO"
}

install_es_ag_rpm()
{
    local OS_SUFFIX="-x86_64"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.rpm"
    local ALGORITHM="512"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"

    log "[install_es_ag_rpm] installing Elastic Agent $STACK_VERSION" "INFO"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_rpm] download location - $DOWNLOAD_URL" "INFO"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_rpm] downloaded Elastic Agent $STACK_VERSION" "INFO"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error validating checksum for Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi

    sudo rpm -vi $PACKAGE
    log "[install_es_ag_rpm] installed Elastic Agent $STACK_VERSION" "INFO"
}

install_es_ag_linux()
{
    local OS_SUFFIX="-linux-x86_64"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.tar.gz"
    local ALGORITHM="512"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"


    log "[install_es_ag_linux] installing Elastic Agent $STACK_VERSION" "INFO"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_linux] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_linux] download location - $DOWNLOAD_URL" "INFO"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_linux] error downloading Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_linux] downloaded Elastic Agent $STACK_VERSION" "INFO"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_linux] error validating checksum for Elastic Agent $STACK_VERSION" "ERROR"
        exit $EXIT_CODE
    fi
    tar xzvf $PACKAGE
    log "[install_es_ag_linux] installed Elastic Agent $STACK_VERSION" "INFO"
}

checkOS

# Enroll Elastic Agent
es_agent_enroll() {
  cloud_hash=$(echo $CLOUD_ID | cut -f2 -d:)
  cloud_tokens=$(echo $cloud_hash | base64 -d -)
  host_port=$(echo $cloud_tokens | cut -f1 -d$)
  local ELASTICSEARCH_URL="https://$(echo $cloud_tokens | cut -f2 -d$).${host_port}"
  local KIBANA_URL="https://$(echo $cloud_tokens | cut -f3 -d$).${host_port}"
  log "[es_agent_enroll] Found ES uri $ELASTICSEARCH_URL and Kibana host $KIBANA_URL" "INFO"
  local ENROLLMENT_TOKEN_ID=""
  local ENROLLMENT_TOKEN=""
  jsonResult=$(curl "${KIBANA_URL}"/api/fleet/enrollment-api-keys  -H 'Content-Type: application/json' -H 'kbn-xsrf: true' -u ${USERNAME}:${PASSWORD} )

      local EXITCODE=$?
      if [ $EXITCODE -ne 0 ]; then
        log "[es_agent_enroll] error calling $KIBANA_URL/api/fleet/enrollment-api-keys in order to retrieve the ENROLLMENT_TOKEN" "ERROR"
        exit $EXITCODE
      fi
      ENROLLMENT_TOKEN_ID=$(echo $jsonResult | jq -r '.list[0].id')
       log "[es_agent_enroll] ENROLLMENT_TOKEN_ID is $ENROLLMENT_TOKEN_ID" "INFO"

      jsonResult=$(curl ${KIBANA_URL}/api/fleet/enrollment-api-keys/$ENROLLMENT_TOKEN_ID \
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
      log "[es_agent_enroll] Enrolling the Elastic Agent to Fleet ${KIBANA_URL}" "INFO"
      elastic-agent enroll  ${KIBANA_URL} $ENROLLMENT_TOKEN -f

}

if [ "$DISTRO_OS" = "DEB" ]; then
    install_es_ag_deb
elif [ "$DISTRO_OS" = "RPM" ]; then
    install_es_ag_rpm
else
  install_es_ag_linux
fi

log "[es_agent_start] enrolling Elastic Agent $STACK_VERSION" "INFO"
es_agent_enroll
log "[es_agent_start] Elastic Agent $STACK_VERSION enrolled" "INFO"
