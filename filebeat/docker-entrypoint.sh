#!/bin/bash
set -e

# This script is the entrypoint to the filebeat Docker container. This will
# verify that all services are running before executing the command provided
# to the docker container.

setDefaults() {
  # Use default ports and hosts if not specified.
  : ${ES_HOST:=localhost}
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

    if [ "$SHIELD" == "true" ]; then
        code=$(curl --write-out "%{http_code}\n" --silent --output /dev/null "http://${ES_HOST}:${ES_PORT}/")

        if [ $code != 401 ]; then
            echo "Shield does not seem to be running"
            exit 1
        fi
    fi
    echo "http://${auth}${ES_HOST}:${ES_PORT}"
}

# Wait for elasticsearch to start. It requires that the status be either
# green or yellow.
waitForElasticsearch() {
  echo -n "Waiting on elasticsearch($(es_url)) to start."
  for ((i=1;i<=60;i++))
  do
    health=$(curl --silent "$(es_url)/_cat/health" | awk '{print $4}')
    if [[ "$health" == "green" ]] || [[ "$health" == "yellow" ]]
    then
      echo
      echo "Elasticsearch is ready!"
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
setDefaults

# Services need to test outputs
# Wait until all services are started
waitForElasticsearch

exec "$@"
