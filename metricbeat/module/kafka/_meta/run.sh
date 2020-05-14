#!/bin/bash

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

${KAFKA_HOME}/bin/kafka-topics.sh --zookeeper=127.0.0.1:2181 --create --partitions 1 --topic test --replication-factor 1

echo "Starting ZooKeeper"
${KAFKA_HOME}/bin/zookeeper-server-start.sh ${KAFKA_HOME}/config/zookeeper.properties &
wait_for_port 2181

echo "Starting Kafka broker"
mkdir -p ${KAFKA_LOGS_DIR}
export KAFKA_OPTS="-Djava.security.auth.login.config=/etc/kafka/server_jaas.conf -javaagent:/opt/jolokia-jvm-1.5.0-agent.jar=port=8779,host=0.0.0.0"
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
wait_for_port 8779


echo "Kafka load status code $?"

# ACLS used to prepare tests
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:producer --operation All --cluster --topic '*' --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:consumer --operation All --cluster --topic '*' --group '*'

# Minimal ACLs required by metricbeat. If this needs to be changed, please update docs too
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Describe --group '*'
${KAFKA_HOME}/bin/kafka-acls.sh --authorizer-properties zookeeper.connect=localhost:2181 --add --allow-principal User:stats --operation Read --topic '*'

touch /tmp/.acls_loaded


# Start a forever producer
{ while sleep 1; do echo message; done } | KAFKA_OPTS="-Djava.security.auth.login.config=/kafka/bin/jaas-kafka-client-producer.conf -javaagent:/opt/jolokia-jvm-1.5.0-agent.jar=port=8775,host=0.0.0.0" \
 ${KAFKA_HOME}/bin/kafka-console-producer.sh --topic test --broker-list localhost:9091 --producer.config ${KAFKA_HOME}/bin/sasl-producer.properties > /dev/null &

wait_for_port 8775

# Start a forever consumer
KAFKA_OPTS="-Djava.security.auth.login.config=/kafka/bin/jaas-kafka-client-consumer.conf -javaagent:/opt/jolokia-jvm-1.5.0-agent.jar=port=8774,host=0.0.0.0" \
 ${KAFKA_HOME}/bin/kafka-console-consumer.sh --topic=test --bootstrap-server=localhost:9091 --consumer.config ${KAFKA_HOME}/bin/sasl-producer.properties > /dev/null &

wait_for_port 8774

# Make sure the container keeps running
tail -f /dev/null
