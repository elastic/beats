#!/bin/bash
set -e

# This script is the entrypoint to the Docker container. This will
# verify that the Elasticsearch is set and that Elasticsearch is running before
# executing the command provided to the docker container.

# Read parameters from the environment and validate them.
readParams() {
  if [ -z "$ES_HOST" ]; then
    echo >&2 'Error: missing required ES_HOST environment variable'
    echo >&2 '  Did you forget to -e ES_HOST=... ?'
    exit 1
  fi

  # Use default ports if not specified.
  : ${ES_PORT:=9200}
}

es_url() {
    local auth

    auth=""
    if [ -n "$ES_USER" ]; then
        auth="$ES_USER"
        if [ -n "$ES_PASS" ]; then
            auth="$auth:$ES_PASS"
        fi
        auth="$auth@"
    fi

    echo "http://${auth}${ES_HOST}:${ES_PORT}"
}

# Wait for elasticsearch to start. It requires that the status be either
# green or yellow.
waitForElasticsearch() {
  echo -n "Waiting on elasticsearch(${ES_HOST}:${ES_PORT}) to start."
  for ((i=1;i<=60;i++))
  do
    health=$(curl --silent "$(es_url)/_cat/health" | awk '{print $4}')
    if [[ "$health" == "green" ]] || [[ "$health" == "yellow" ]]
    then
      echo
      echo "Elasticsearch($(es_url)) is ready!"
      return 0
    fi

    echo -n '.'
    sleep 1
  done

  echo
  echo >&2 'Elasticsearch is not running or is not healthy.'
  echo >&2 "Address: $(es_url)"
  echo >&2 "$health"
  exit 1
}

# Main

readParams
waitForElasticsearch
exec "$@"
