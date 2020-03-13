#!/bin/bash

echo "==================================================================================="
echo "==== Kerberos Client =============================================================="
echo "==================================================================================="
KADMIN_PRINCIPAL_FULL=$KADMIN_PRINCIPAL@$REALM

echo "REALM: $REALM"
echo "KADMIN_PRINCIPAL_FULL: $KADMIN_PRINCIPAL_FULL"
echo "KADMIN_PASSWORD: $KADMIN_PASSWORD"
echo ""

function kadminCommand {
    kadmin -p $KADMIN_PRINCIPAL_FULL -w $KADMIN_PASSWORD -q "$1"
}

echo "==================================================================================="
echo "==== /etc/krb5.conf ==============================================================="
echo "==================================================================================="
tee /etc/krb5.conf <<EOF
[libdefaults]
	default_realm = $REALM

[realms]
	$REALM = {
		kdc = kerberos_kdc
		admin_server = kerberos_kdc
	}
EOF
echo ""

echo "==================================================================================="
echo "==== Testing ======================================================================"
echo "==================================================================================="
until kadminCommand "list_principals $KADMIN_PRINCIPAL_FULL"; do
  >&2 echo "KDC is unavailable - sleeping 1 sec"
  sleep 1
done
echo "KDC and Kadmin are operational"
echo ""

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

# Init Kerberos principal
echo "Runnning kinit"

kinit -k -t /etc/security/keytabs/kafka.keytab $KADMIN_PRINCIPAL_FULL

echo "Starting ZooKeeper"

_JAVA_OPTIONS="${_JAVA_OPTIONS} -Djava.security.auth.login.config=/etc/kafka/zookeeper_jaas.conf" \
${KAFKA_HOME}/bin/zookeeper-server-start.sh ${KAFKA_HOME}/config/zookeeper.properties &
wait_for_port 2181


echo "Starting Kafka broker"
mkdir -p ${KAFKA_LOGS_DIR}
_JAVA_OPTIONS="${_JAVA_OPTIONS} -Djava.security.auth.login.config=/etc/kafka/server_jaas.conf" \
${KAFKA_HOME}/bin/kafka-server-start.sh ${KAFKA_HOME}/config/server.properties \
    --override delete.topic.enable=true --override advertised.host.name=${KAFKA_ADVERTISED_HOST} \
    --override listeners=PLAINTEXT://kafka_kerberos:9092,SASL_SSL://kafka_kerberos:9093 \
    --override advertised.listeners=PLAINTEXT://${ADVERTISED_HOST}:9092,SASL_SSL://${ADVERTISED_HOST}:9093 \
    --override logs.dir=${KAFKA_LOGS_DIR} --override log.flush.interval.ms=200 \
    --override num.partitions=3 \
    --override sasl.enabled.mechanisms=GSSAPI \
    --override sasl.mechanism.inter.broker.protocol=GSSAPI \
    --override sasl.kerberos.service.name="kafka" &

wait_for_port 9092

echo "Kafka load status code $?"

# Make sure the container keeps running
tail -f /dev/null
