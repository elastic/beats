FROM openjdk:8-jre-alpine3.8

ARG KAFKA_VERSION=2.1.0

ENV KAFKA_HOME /kafka
ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true"
ENV TERM=linux

RUN apk add -u bash

RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && \
    wget -q -O $INSTALL_DIR/kafka.tgz "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1 && \
    rm -f ${INSTALL_DIR}/kafka.tgz

ADD run.sh /run.sh
ADD healthcheck.sh /healthcheck.sh

EXPOSE 9092
EXPOSE 2181

# Healthcheck creates an empty topic foo. As soon as a topic is created, it assumes broke is available
HEALTHCHECK --interval=1s --retries=90 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
