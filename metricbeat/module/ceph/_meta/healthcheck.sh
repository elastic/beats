#!/bin/bash

KEYS_JSON=/opt/ceph-container/sree/static/restful-list-keys.json

if [[ ! -f "${KEYS_JSON}" ]]; then
  ceph restful list-keys | grep demo >/dev/null
  if [[ $? -eq 0 ]]; then
    ceph restful list-keys > ${KEYS_JSON}
  else
    exit 1
  fi
fi

ceph health | grep HEALTH_OK && curl -s localhost:5000 >/dev/null
