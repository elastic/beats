#!/usr/bin/env bash
set -euo pipefail
script_path=$(dirname $(realpath -s $0))
source $script_path/helper.sh




write_status() {
  get_status_location
  if [[ "$STATUS_FOLDER" != "" ]]; then
    if [ -n "$(ls -A "$STATUS_FOLDER" 2>/dev/null)" ]; then
      status_files_path="$STATUS_FOLDER/*.status"
      latest_status_file=$(ls $status_files_path -A1 | sort -V | tail -1)
      echo $latest_status_file
    else
      echo "here"
      echo $'First line.\nSecond line.\nThird line.' > "$STATUS_FOLDER"/1.status
    fi
  fi
}

write_status
