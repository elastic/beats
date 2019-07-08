FROM debian:stretch

ARG TAG=2.1.1

ENV KAFKA_HOME /kafka

ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true"
ENV TERM=linux

RUN apt-get update && apt-get install -y curl openjdk-8-jre-headless netcat dnsutils

RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && curl -s -o $INSTALL_DIR/kafka.tgz \
    "https://archive.apache.org/dist/kafka/${TAG}/kafka_2.11-${TAG}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1

ADD kafka_server_jaas.conf /etc/kafka/server_jaas.conf
ADD run.sh /run.sh
ADD healthcheck.sh /healthcheck.sh

EXPOSE 9092
EXPOSE 2181

# Healthcheck creates an empty topic foo. As soon as a topic is created, it assumes broke is available
HEALTHCHECK --interval=1s --retries=90 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
