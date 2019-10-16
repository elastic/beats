#!/bin/bash

ZOOKEEPER_HOST=${ZOOKEEPER_HOST:-zookeeper}

if [ -n "$KAFKA_ADVERTISED_HOST_AUTO" ]; then
	KAFKA_ADVERTISED_HOST=$(dig +short $HOSTNAME):9092
fi

# Check if KAFKA_ADVERTISED_HOST is set
# if not wait to read it from file
if [ -z "$KAFKA_ADVERTISED_HOST" ]; then
       echo "SERVICE_HOST needed, will wait for it on /run/compose_env"
       while true; do
               if [ -f /run/compose_env ]; then
                       source /run/compose_env
                       KAFKA_ADVERTISED_HOST=$SERVICE_HOST
               fi
               if [ -n "$KAFKA_ADVERTISED_HOST" ]; then
                       # Remove it so it is not reused
                       > /run/compose_env
                       break
               fi
               sleep 1
       done
fi

wait_for() {
    count=20
    host=$1
    port=$2
    while ! nc -z $host $port && [[ $count -ne 0 ]]; do
        count=$(( $count - 1 ))
        [[ $count -eq 0 ]] && return 1
        sleep 0.5
    done
    # just in case, one more time
    nc -z localhost $port
}


echo "Waiting for ZooKeeper"
wait_for $ZOOKEEPER_HOST 2181

echo "Starting Kafka broker"
mkdir -p ${KAFKA_LOGS_DIR}
export KAFKA_OPTS=-Djava.security.auth.login.config=/etc/kafka/server_jaas.conf
${KAFKA_HOME}/bin/kafka-server-start.sh ${KAFKA_HOME}/config/server.properties \
    --override zookeeper.connect=$ZOOKEEPER_HOST:2181 \
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

wait_for localhost 9092

echo "Kafka load status code $?"

# ACLS used to prepare tests
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=${ZOOKEEPER_HOST}:2181 --add --allow-principal User:producer --operation All --cluster --topic '*' --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=${ZOOKEEPER_HOST}:2181 --add --allow-principal User:consumer --operation All --cluster --topic '*' --group '*'

# Minimal ACLs required by metricbeat. If this needs to be changed, please update docs too
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=${ZOOKEEPER_HOST}:2181 --add --allow-principal User:stats --operation Describe --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=${ZOOKEEPER_HOST}:2181 --add --allow-principal User:stats --operation Read --topic '*'

touch /tmp/.acls_loaded

# Make sure the container keeps running
tail -f /dev/null
