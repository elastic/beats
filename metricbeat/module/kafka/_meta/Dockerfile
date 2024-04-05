FROM debian:stretch-slim

ARG KAFKA_VERSION

ENV KAFKA_HOME=/kafka \
    KAFKA_LOGS_DIR=/kafka-logs \
    _JAVA_OPTIONS="-Djava.net.preferIPv4Stack=true" \
    TERM=linux

# Update apt sources and install necessary packages
RUN sed -i 's/deb.debian.org/archive.debian.org/g' /etc/apt/sources.list \
    && sed -i 's|security.debian.org|archive.debian.org/|g' /etc/apt/sources.list \
    && sed -i '/stretch-updates/d' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends curl openjdk-8-jre-headless \
    && rm -rf /var/lib/apt/lists/*

# Install Kafka
RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && \
    curl -J -L -s -f --show-error -o $INSTALL_DIR/kafka.tgz \
        "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1 \
    && rm -f ${INSTALL_DIR}/kafka.tgz

# Install Jolokia
RUN curl -J -L -s -f --show-error -o /opt/jolokia-jvm-1.5.0-agent.jar \
    http://search.maven.org/remotecontent\?filepath\=org/jolokia/jolokia-jvm/1.5.0/jolokia-jvm-1.5.0-agent.jar

ADD kafka_server_jaas.conf /etc/kafka/server_jaas.conf
ADD jaas-kafka-client-producer.conf sasl-producer.properties jaas-kafka-client-consumer.conf sasl-consumer.properties /kafka/bin/
ADD run.sh healthcheck.sh /

# Make scripts executable
RUN chmod +x /run.sh /healthcheck.sh

# Expose necessary ports
EXPOSE 9092 2181 8779 8775 8774

# Healthcheck creates an empty topic foo. As soon as a topic is created, it assumes broker is available
HEALTHCHECK --interval=1s --retries=700 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
