CLOUD_ID=${CLOUD_ID-""}
USERNAME=${USERNAME-""}
PASSWORD=${PASSWORD-""}

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


log()
{
    echo \[$(date +%d%m%Y-%H:%M:%S)\]  "$2" "$1"
    echo \[$(date +%d%m%Y-%H:%M:%S)\]  "$2" "$1" >> /var/log/es-agent-install.log
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
