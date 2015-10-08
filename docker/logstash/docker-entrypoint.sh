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
  : ${REDIS_PORT:=6379}
}

# Wait for elasticsearch to start. It requires that the status be either
# green or yellow.
waitForElasticsearch() {
  echo -n 'Waiting on elasticsearch to start.'
  for ((i=1;i<=30;i++))
  do
    health=$(curl --silent "http://${ES_HOST}:${ES_PORT}/_cat/health" | awk '{print $4}')
    if [[ "$health" == "green" ]] || [[ "$health" == "yellow" ]]
    then
      echo
      echo "Elasticsearch is ready!"
      return 0
    fi

    ((i++))
    echo -n '.'
    sleep 1
  done

  echo
  echo >&2 'Elasticsearch is not running or is not healthy.'
  echo >&2 "Address: ${ES_HOST}:${ES_PORT}"
  echo >&2 "$health"
  exit 1
}

updateConfigFile() {
    sed -i -e "s/host.*/host => \"$ES_HOST\"/" /logstash.conf
}

# Main
readParams
updateConfigFile
waitForElasticsearch
exec "$@"
