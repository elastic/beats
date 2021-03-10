#!/usr/bin/env bash
set -euo pipefail


LINUX_CERT_PATH="/var/lib/waagent"



decrypt1() {
  cert_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.crt"
  echo $cert_path
  private_key_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    dec=$(openssl cms -decrypt -in <(echo "$2" | base64 --decode) -inkey $private_key_path -recip $cert_path -inform dem)
    echo $dec
     CLOUD_ID=$(echo $dec | jq -r '.password')
     echo $CLOUD_ID
    else
    echo "ERROR" "[decrypt] Decryption failed. Could not find certificates"
    exit 1
fi
}

encrypt() {
  cert_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.crt"
  echo $cert_path
  private_key_path="/mnt/c/Users/maria/Downloads/test/waagent/$1.prv"
  if [[ -f "$cert_path" ]] && [[ -f "$private_key_path" ]]; then
    echo "files exists."
    openssl cms -encrypt -in <(echo "$2") -inkey $private_key_path -recip $cert_path -inform dem
    else
    echo "ERROR" "[decrypt] Decryption failed. Could not find certificates"
    exit 1
fi
}


