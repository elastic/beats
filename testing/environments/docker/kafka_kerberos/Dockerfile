FROM debian:stretch
MAINTAINER Simão Martins "simao.martins@tecnico.ulisboa.pt"

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get -qq update \
  && apt-get -qq install \
      apt-transport-https \
      curl \
      krb5-user \
      locales \
      netcat \
      openjdk-8-jre-headless \
  && apt-get -qq clean

RUN locale-gen "en_US.UTF-8"
RUN echo "LC_ALL=\"en_US.UTF-8\"" >> /etc/default/locale

# Kerberos variables
ENV REALM ELASTIC
ENV KADMIN_PRINCIPAL kafka
ENV KADMIN_PASSWORD changeme

ENV KAFKA_HOME /kafka
# The advertised host is kafka. This means it will not work if container is started locally and connected from localhost to it
ENV KAFKA_ADVERTISED_HOST kafka
ENV KAFKA_LOGS_DIR="/kafka-logs"
ENV KAFKA_VERSION 2.1.1
ENV _JAVA_OPTIONS "-Djava.net.preferIPv4Stack=true -Djava.security.krb5.conf=/etc/kafka/krb5.conf"
ENV TERM=linux

RUN mkdir -p ${KAFKA_LOGS_DIR} && mkdir -p ${KAFKA_HOME} && curl -s -o $INSTALL_DIR/kafka.tgz \
    "http://mirror.easyname.ch/apache/kafka/${KAFKA_VERSION}/kafka_2.11-${KAFKA_VERSION}.tgz" && \
    tar xzf ${INSTALL_DIR}/kafka.tgz -C ${KAFKA_HOME} --strip-components 1

ADD kafka_server_jaas.conf /etc/kafka/server_jaas.conf
ADD zookeeper_jaas.conf /etc/kafka/zookeeper_jaas.conf
ADD kafka.keytab /etc/security/keytabs/kafka.keytab
ADD krb5.conf /etc/kafka/krb5.conf
ADD run.sh /run.sh
ADD healthcheck.sh /healthcheck.sh

EXPOSE 9092
EXPOSE 2181

# Healthcheck creates an empty topic foo. As soon as a topic is created, it assumes broke is available
HEALTHCHECK --interval=1s --retries=600 CMD /healthcheck.sh

ENTRYPOINT ["/run.sh"]
