FROM debian:stretch

ENV KAFKA_HOME /kafka
# The advertised host is kafka. This means it will not work if container is started locally and connected from localhost to it
ENV KAFKA_ADVERTISED_HOST kafka
ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV KAFKA_VERSION 2.2.2
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true"
ENV TERM=linux

RUN apt-get update && apt-get install -y curl openjdk-8-jre-headless netcat

RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && \
    curl -J -L -s -f -o - https://github.com/kadwanev/retry/releases/download/1.0.1/retry-1.0.1.tar.gz | tar xfz - -C /usr/local/bin && \
    retry --min 1 --max 180 -- curl -J -L -s -f --show-error -o $INSTALL_DIR/kafka.tgz \
        "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1

ADD run.sh /run.sh
ADD healthcheck.sh /healthcheck.sh
ADD certs/broker.keystore.jks /broker.keystore.jks
ADD certs/client.truststore.jks /broker.truststore.jks

EXPOSE 9092
EXPOSE 9093
EXPOSE 2181

# healthcheck.sh tries to create and delete an empty kafka topic (the topic
# string is  based on the timestamp), and reports healthy if topic creation
# was successful.
# With these parameters, Docker will consider the container unhealthy if the
# Kafka server is unresponsive for 3 minutes.
HEALTHCHECK --start-period=10s --interval=5s --timeout=5s --retries=36 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
