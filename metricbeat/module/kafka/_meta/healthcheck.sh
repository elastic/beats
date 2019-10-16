#!/bin/bash

ZOOKEEPER_HOST=${ZOOKEEPER_HOST:-zookeeper}

[ -f /tmp/.acls_loaded ] || exit 1

TOPIC="foo-`date '+%s-%N'`"

${KAFKA_HOME}/bin/kafka-topics.sh --zookeeper=${ZOOKEEPER_HOST}:2181 --create --partitions 1 --topic "${TOPIC}" --replication-factor 1
rc=$?
if [[ $rc != 0 ]]; then
	exit $rc
fi

${KAFKA_HOME}/bin/kafka-topics.sh --zookeeper=${ZOOKEEPER_HOST}:2181 --delete --topic "${TOPIC}"
exit 0
