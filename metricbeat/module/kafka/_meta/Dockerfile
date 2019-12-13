FROM debian:stretch

ARG KAFKA_VERSION

ENV KAFKA_HOME /kafka

ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true"
ENV TERM=linux

RUN apt-get update && apt-get install -y curl openjdk-8-jre-headless netcat dnsutils

RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && \
    curl -J -L -s -f -o - https://github.com/kadwanev/retry/releases/download/1.0.1/retry-1.0.1.tar.gz | tar xfz - -C /usr/local/bin && \
    retry --min 1 --max 180 -- curl -J -L -s -f --show-error -o $INSTALL_DIR/kafka.tgz \
        "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1

RUN retry --min 1 --max 180 -- curl -J -L -s -f --show-error -o /opt/jolokia-jvm-1.5.0-agent.jar \
    http://search.maven.org/remotecontent\?filepath\=org/jolokia/jolokia-jvm/1.5.0/jolokia-jvm-1.5.0-agent.jar

ADD kafka_server_jaas.conf /etc/kafka/server_jaas.conf
ADD jaas-kafka-client-producer.conf /kafka/bin/jaas-kafka-client-producer.conf
ADD sasl-producer.properties /kafka/bin/sasl-producer.properties
ADD jaas-kafka-client-consumer.conf /kafka/bin/jaas-kafka-client-consumer.conf
ADD sasl-consumer.properties /kafka/bin/sasl-consumer.properties
ADD run.sh /run.sh
ADD healthcheck.sh /healthcheck.sh

EXPOSE 9092
EXPOSE 2181
EXPOSE 8779
EXPOSE 8775
EXPOSE 8774

# Healthcheck creates an empty topic foo. As soon as a topic is created, it assumes broker is available
HEALTHCHECK --interval=1s --retries=700 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
