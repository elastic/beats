#!/bin/bash

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

# create a user beats with password KafkaTest, for use in client SASL authentication
/kafka/bin/kafka-configs.sh \
	--zookeeper localhost:2181 \
	--alter --add-config 'SCRAM-SHA-512=[password=KafkaTest]' \
	--entity-type users \
	--entity-name beats

# Start Kafka with three listeners. The INSIDE listener makes Kafka reachable inside of docker
# networks when the container hostname matches KAFKA_ADVERTISED_HOST. The OUTSIDE and SASL_SSL both
# bind to localhost and are reachable from the host machine on the loopback interface.
echo "Starting Kafka broker"
mkdir -p ${KAFKA_LOGS_DIR}
${KAFKA_HOME}/bin/kafka-server-start.sh ${KAFKA_HOME}/config/server.properties \
    --override delete.topic.enable=true \
    --override listener.security.protocol.map=INSIDE:PLAINTEXT,OUTSIDE:PLAINTEXT,SASL_SSL:SASL_SSL \
    --override listeners=INSIDE://0.0.0.0:9092,OUTSIDE://0.0.0.0:9094,SASL_SSL://0.0.0.0:9093 \
    --override advertised.listeners=INSIDE://${KAFKA_ADVERTISED_HOST}:9092,OUTSIDE://localhost:9094,SASL_SSL://localhost:9093 \
    --override inter.broker.listener.name=INSIDE \
    --override sasl.enabled.mechanisms=SCRAM-SHA-512 \
    --override listener.name.sasl_ssl.scram-sha-512.sasl.jaas.config="org.apache.kafka.common.security.scram.ScramLoginModule required;" \
    --override logs.dir=${KAFKA_LOGS_DIR} \
    --override log4j.logger.kafka=DEBUG,kafkaAppender \
    --override log.flush.interval.ms=200 \
    --override num.partitions=3 \
    --override ssl.keystore.location=/broker.keystore.jks \
    --override ssl.keystore.password=KafkaTest \
    --override ssl.truststore.location=/broker.truststore.jks \
    --override ssl.truststore.password=KafkaTest &

wait_for_port 9092

echo "Kafka load status code $?"

# Make sure the container keeps running
tail -f /dev/null
