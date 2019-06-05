#!/bin/bash

KAFKA_ADVERTISED_HOST=$(dig +short $HOSTNAME):9092

wait_for_port() {
    count=20
    port=$1
    while ! nc -z localhost $port && [[ $count -ne 0 ]]; do
        count=$(( $count - 1 ))
        [[ $count -eq 0 ]] && return 1
        sleep 0.5
    done
    # just in case, one more time
    nc -z localhost $port
}

echo "Starting ZooKeeper"
${KAFKA_HOME}/bin/zookeeper-server-start.sh ${KAFKA_HOME}/config/zookeeper.properties &
wait_for_port 2181

echo "Starting Kafka broker"
mkdir -p ${KAFKA_LOGS_DIR}
export KAFKA_OPTS=-Djava.security.auth.login.config=/etc/kafka/server_jaas.conf
${KAFKA_HOME}/bin/kafka-server-start.sh ${KAFKA_HOME}/config/server.properties \
    --override authorizer.class.name=kafka.security.auth.SimpleAclAuthorizer \
    --override super.users=User:admin \
    --override sasl.enabled.mechanisms=PLAIN \
    --override sasl.mechanism.inter.broker.protocol=PLAIN \
    --override delete.topic.enable=true \
    --override listeners=INSIDE://localhost:9091,OUTSIDE://0.0.0.0:9092 \
    --override advertised.listeners=INSIDE://localhost:9091,OUTSIDE://$KAFKA_ADVERTISED_HOST \
    --override listener.security.protocol.map=INSIDE:SASL_PLAINTEXT,OUTSIDE:SASL_PLAINTEXT \
    --override inter.broker.listener.name=INSIDE \
    --override logs.dir=${KAFKA_LOGS_DIR} &

wait_for_port 9092

echo "Kafka load status code $?"

# ACLS used to prepare tests
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:producer --operation All --cluster --topic '*' --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:consumer --operation All --cluster --topic '*' --group '*'

# Minimal ACLs required by metricbeat. If this needs to be changed, please update docs too
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Describe --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Read --topic '*'

touch /tmp/.acls_loaded

# Make sure the container keeps running
tail -f /dev/null
