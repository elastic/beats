#!/bin/bash

set -e

until curl -f -s "http://elasticsearch:9200/_license"; do
  echo "Elasticsearch not available yet".
  sleep 1
done

/usr/local/bin/docker-entrypoint.sh "$@"
