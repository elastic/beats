#!/bin/bash
set -e

# This script is the entrypoint to the libbeat Docker container. This will
# verify that the Elasticsearch and Redis environment variables are set
# and that Elasticsearch is running before executing the command provided
# to the docker container.

# Read parameters from the environment and validate them.
checkHost() {
    if [ -z "$$1" ]; then
        echo >&2 'Error: missing required $1 environment variable'
        echo >&2 '  Did you forget to -e $1=... ?'
        exit 1
    fi
}

readParams() {
  checkHost "ES_HOST"
  checkHost "REDIS_HOST"
  checkHost "LS_HOST"

  # Use default ports if not specified.
  : ${ES_PORT:=9200}
  : ${REDIS_PORT:=6379}
  : ${LS_TCP_PORT:=12345}
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
  echo -n "Waiting on elasticsearch($(es_url)) to start."
  for ((i=1;i<=30;i++))
  do
    health=$(curl --silent "$(es_url)/_cat/health" | awk '{print $4}')
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
  echo >&2 "Address: $(es_url)"
  echo >&2 "$health"
  exit 1
}

waitForLogstash() {
    echo -n "Waiting for logstash(${LS_HOST}:${LS_TCP_PORT}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${LS_HOST} ${LS_TCP_PORT} 2>/dev/null; then
            echo
            echo "Logstash is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 'Logstash is not available'
    echo >&2 "Address: ${LS_HOST}:${LS_TCP_PORT} and ${LS_HOST}:${LS_TLS_PORT}"
}

# Main
readParams
waitForElasticsearch
waitForLogstash
exec "$@"
