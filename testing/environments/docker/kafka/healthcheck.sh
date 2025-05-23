#!/bin/bash

TOPIC="foo-`date '+%s-%N'`"

${KAFKA_HOME}/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --partitions 1 --topic "${TOPIC}" --replication-factor 1
rc=$?
if [[ $rc != 0 ]]; then
	exit $rc
fi

${KAFKA_HOME}/bin/kafka-topics.sh --bootstrap-server localhost:9092  --delete --topic "${TOPIC}"
exit 0
