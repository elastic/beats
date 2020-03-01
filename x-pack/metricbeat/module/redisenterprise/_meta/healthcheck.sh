#!/bin/bash

set -e

CHECK_CLUSTER_CREATED=/opt/redislabs/config/check_cluster_created
CHECK_DATABASE_CREATED=/opt/redislabs/config/check_database_created

if [[ ! -f "${CHECK_CLUSTER_CREATED}" ]]; then
  rladmin cluster create name cluster.local username cihan@redislabs.com password redislabs123
  touch ${CHECK_CLUSTER_CREATED}
fi

if [[ ! -f "${CHECK_DATABASE_CREATED}" ]]; then
  curl -s -k -u "cihan@redislabs.com:redislabs123" --request POST \
    --url "https://localhost:9443/v1/bdbs" \
    --header 'content-type: application/json' \
    --data '{"name":"db1","type":"redis","memory_size":102400,"port":12000}'
  touch ${CHECK_DATABASE_CREATED}
fi

curl -s --insecure https://127.0.0.1:8070 >/dev/null
