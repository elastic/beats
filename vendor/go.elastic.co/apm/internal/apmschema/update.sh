#!/usr/bin/env bash

set -ex

BRANCH=master

FILES=( \
    "error.json" \
    "metadata.json" \
    "metricset.json" \
    "span.json" \
    "transaction.json" \
)

for i in "${FILES[@]}"; do
  o=jsonschema/$i
  curl -sf https://raw.githubusercontent.com/elastic/apm-server/${BRANCH}/docs/spec/v2/${i} --compressed -o $o
done
