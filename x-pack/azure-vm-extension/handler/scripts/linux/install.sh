#!/usr/bin/env bash
set -euo pipefail

DISTRO_OS=""

checkOS()
{
  if dpkg -S /bin/ls >/dev/null 2>&1
then
  DISTRO_OS="DEB"
  log "[checkOS] distro is $DISTRO_OS"
elif rpm -q -f /bin/ls >/dev/null 2>&1
then
  DISTRO_OS="RPM"
   log "[checkOS] distro is $DISTRO_OS"
else
  echo "Don't know this package system (neither RPM nor DEB)."
  exit 1
fi
}

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

checkOS

if [ "$DISTRO_OS" = "DEB" ]; then
    install_es_ag_deb
elif [ "$DISTRO_OS" = "RPM" ]; then
    install_es_ag_rpm
else
  echo "Don't know this package system (neither RPM nor DEB)."
fi

