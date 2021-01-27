#!/usr/bin/env bash
set -euo pipefail

log()
{
    echo \[$(date +%d%m%Y-%H:%M:%S)\] "$1"
    echo \[$(date +%d%m%Y-%H:%M:%S)\] "$1" >> /var/log/es-agent-install.log
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

    log "[install_es_ag_deb] installing Elastic Agent $STACK_VERSION"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_deb] download location - $DOWNLOAD_URL"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_deb] error downloading Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_deb] downloaded Elastic Agent $STACK_VERSION"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_deb] error validating checksum for Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi

    sudo dpkg -i $PACKAGE
    log "[install_es_ag_deb] installed Elastic Agent $STACK_VERSION"
}

install_es_ag_rpm()
{
    local OS_SUFFIX="-x86_64"
    local PACKAGE="elastic-agent-${STACK_VERSION}${OS_SUFFIX}.rpm"
    local ALGORITHM="512"
    local SHASUM="$PACKAGE.sha$ALGORITHM"
    local DOWNLOAD_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}"
    local SHASUM_URL="https://artifacts.elastic.co/downloads/beats/elastic-agent/${PACKAGE}.sha512"

    log "[install_es_ag_rpm] installing Elastic Agent $STACK_VERSION"
    wget --retry-connrefused --waitretry=1 "$SHASUM_URL" -O "$SHASUM"
    local EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION sha$ALGORITHM checksum"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_rpm] download location - $DOWNLOAD_URL"
    wget --retry-connrefused --waitretry=1 "$DOWNLOAD_URL" -O $PACKAGE
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error downloading Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi
    log "[install_es_ag_rpm] downloaded Elastic Agent $STACK_VERSION"

    #checkShasum $PACKAGE $SHASUM
    EXIT_CODE=$?
    if [[ $EXIT_CODE -ne 0 ]]; then
        log "[install_es_ag_rpm] error validating checksum for Elastic Agent $STACK_VERSION"
        exit $EXIT_CODE
    fi

    sudo rpm -vi $PACKAGE
    log "[install_es_ag_rpm] installed Elastic Agent $STACK_VERSION"
}

checkOS

if [ "$DISTRO_OS" = "DEB" ]; then
    install_es_ag_deb
elif [ "$DISTRO_OS" = "RPM" ]; then
    install_es_ag_rpm
else
  echo "Don't know this package system (neither RPM nor DEB)."
fi

