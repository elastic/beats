#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh

LINUX_CERT_PATH="/var/lib/waagent"


decrypt() {
  get_protected_settings
  get_thumbprint
  #PROTECTED_SETTINGS
  THUMBPRINT="75944CEE9BE769DD92EC5F39B66F73B9EBC45A2B"
  cert_path="$LINUX_CERT_PATH/$THUMBPRINT.crt"
  private_key_path="$LINUX_CERT_PATH}/$THUMBPRINT.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    echo "$FILE exists."

    encrypted=$(echo "$PROTECTED_SETTINGS" | base64 --decode)
    else
    log "ERROR" "[decrypt] Decryption failed. Could not find certificates"
    exit 1
fi
}

