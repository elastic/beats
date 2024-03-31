FROM debian:buster

ENV KAFKA_HOME /kafka
# Controls the hostname advertised within the Docker network, should generally match the container
# name in a docker-compose file.
ENV KAFKA_ADVERTISED_HOST kafka

ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV KAFKA_VERSION 2.2.2
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true"
ENV TERM=linux

# Install required dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    openjdk-11-jre-headless \
    netcat-openbsd && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*


RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && \
   curl -J -L -s -f --show-error -o $INSTALL_DIR/kafka.tgz \
        "https://archive.apache.org/dist/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1 \
    && rm ${INSTALL_DIR}/kafka.tgz

ADD run.sh healthcheck.sh /
ADD certs/broker.keystore.jks /broker.keystore.jks
ADD certs/client.truststore.jks /broker.truststore.jks

# Expose ports
EXPOSE 9092 9093 2181

# Make scripts executable
RUN chmod +x /run.sh /healthcheck.sh

# healthcheck.sh tries to create and delete an empty kafka topic (the topic
# string is  based on the timestamp), and reports healthy if topic creation
# was successful.
# With these parameters, Docker will consider the container unhealthy if the
# Kafka server is unresponsive for 3 minutes.
HEALTHCHECK --start-period=10s --interval=5s --timeout=5s --retries=36 CMD /healthcheck.sh

ENTRYPOINT ["./run.sh"]
