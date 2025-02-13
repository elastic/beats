#!/bin/bash

[ -f /tmp/.acls_loaded ] || exit 1

TOPIC="foo-`date '+%s-%N'`"

${KAFKA_HOME}/bin/kafka-topics.sh --bootstrap-server localhost:9091 --command-config ${KAFKA_HOME}/bin/adminclient.properties --create --partitions 1 --topic "${TOPIC}" --replication-factor 1
rc=$?
if [[ $rc != 0 ]]; then
	exit $rc
fi

${KAFKA_HOME}/bin/kafka-topics.sh --bootstrap-server localhost:9091 --command-config ${KAFKA_HOME}/bin/adminclient.properties --delete --topic "${TOPIC}"
exit 0
