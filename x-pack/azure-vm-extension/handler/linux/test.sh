#!/usr/bin/env bash
set -euo pipefail

get_logs_location()
{
  SCRIPT=$(readlink -f "$0")
  echo $SCRIPT
  ES_EXT_DIR=$(dirname "$SCRIPT")
  echo $cmd_path
  ES_EXT_DIR=$(cd "$( dirname "${cmd_path}" )" >/dev/null 2>&1 && pwd)
    echo $ES_EXT_DIR
   if [ -e $ES_EXT_DIR/HandlerEnvironment.json ]; then
    LOGS_FOLDER=$(jq -r '.[0].handlerEnvironment.logFolder' $ES_EXT_DIR/HandlerEnvironment.json)
  else
    exit 1
  fi
}

get_logs_location
echo $LOGS_FOLDER
