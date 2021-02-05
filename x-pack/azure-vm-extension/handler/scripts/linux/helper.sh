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
