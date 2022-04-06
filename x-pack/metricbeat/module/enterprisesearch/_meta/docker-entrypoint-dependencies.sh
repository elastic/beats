#!/bin/bash

set -e

# Bash is not good at retrieving env variables with dot; parse them with grep from env
export USERNAME=`env | grep elasticsearch.\\username= | cut -d= -f2-`
export PASSWORD=`env | grep elasticsearch.\\password= | cut -d= -f2-`

until curl -u $USERNAME:$PASSWORD -f -s "http://elasticsearch:9200/_license"; do
  echo "Elasticsearch not available yet".
  sleep 1
done

/usr/local/bin/docker-entrypoint.sh "$@"
