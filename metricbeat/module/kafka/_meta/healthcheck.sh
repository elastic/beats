#!/bin/bash

[ -f /tmp/.acls_loaded ] || exit 1

TOPIC="foo-`date '+%s-%N'`"

${KAFKA_HOME}/bin/kafka-topics.sh --zookeeper=127.0.0.1:2181 --create --partitions 1 --topic "${TOPIC}" --replication-factor 1
rc=$?
if [[ $rc != 0 ]]; then
	exit $rc
fi

${KAFKA_HOME}/bin/kafka-topic.sh --zookeeper=127.0.0.1:2181 --delete --topic "${TOPIC}"
exit 0
