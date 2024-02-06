#!/usr/bin/env bash

set -euo pipefail

unset_secrets () {
  for var in $(printenv | sed 's;=.*;;' | sort); do
    if [[ "$var" == *_SECRET || "$var" == *_TOKEN ]]; then
      unset "$var"
    fi
  done
}

google_cloud_logout_active_account() {
  local active_account=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null)
  if [[ -n "$active_account" && -n "${GOOGLE_APPLICATION_CREDENTIALS+x}" ]]; then
    echo "Logging out from GCP for active account"
    gcloud auth revoke $active_account > /dev/null 2>&1
  else
    echo "No active GCP accounts found."
  fi
  if [ -n "${GOOGLE_APPLICATION_CREDENTIALS+x}" ]; then
    unset GOOGLE_APPLICATION_CREDENTIALS
    cleanup
  fi
}

cleanup() {
  echo "Deleting temporary files..."
  rm -rf ${BIN}/${TMP_FOLDER}.*
  echo "Done."
}
