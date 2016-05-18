#!/bin/bash
set -e

# This script is the entrypoint to the libbeat Docker container. This will
# verify that all services are running before executing the command provided
# to the docker container.

setDefaults() {
  # Use default ports and hosts if not specified.
  : ${ES_HOST:=localhost}
  : ${ES_PORT:=9200}
  : ${REDIS_HOST:=localhost}
  : ${REDIS_PORT:=6379}
  : ${LS_HOST:=localhost}
  : ${LS_TCP_PORT:=5044}
  : ${KAFKA_HOST:=localhost}
  : ${KAFKA_PORT:=9092}
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

    if [ $SHIELD == "true" ]; then
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

# Wait for. Params: host, port, service
waitFor() {
    echo -n "Waiting for ${3}(${1}:${2}) to start."
    for ((i=1; i<=90; i++)) do
        if nc -vz ${1} ${2} 2>/dev/null; then
            echo
            echo "${3} is ready!"
            return 0
        fi

        ((i++))
        echo -n '.'
        sleep 1
    done

    echo
    echo >&2 '${3} is not available'
    echo >&2 "Address: ${1}:${2}"
}

# Main
setDefaults

# Services need to test outputs
# Wait until all services are started
waitForElasticsearch
waitFor ${KAFKA_HOST} ${KAFKA_PORT} Kafka
waitFor ${LS_HOST} ${LS_TCP_PORT} Logstash
waitFor ${REDIS_HOST} ${REDIS_PORT} Redis

exec "$@"
