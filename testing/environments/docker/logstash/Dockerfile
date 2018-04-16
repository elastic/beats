FROM centos:7
LABEL maintainer "Elastic Docker Team <docker@elastic.co>"

# Beats variables.
ARG DOWNLOAD_URL
ARG ELASTIC_VERSION
ARG CACHE_BUST=1
ARG IMAGE_FLAVOR=x-pack

# Install Java and the "which" command, which is needed by Logstash's shell
# scripts.
RUN yum update -y && yum install -y java-1.8.0-openjdk-devel which && \
    yum clean all

# Provide a non-root user to run the process.
RUN groupadd --gid 1000 logstash && \
    adduser --uid 1000 --gid 1000 \
      --home-dir /usr/share/logstash --no-create-home \
      logstash

# Add Logstash itself.
COPY download.sh /download.sh
RUN /download.sh $DOWNLOAD_URL $ELASTIC_VERSION $CACHE_BUST && rm /download.sh
RUN tar zxf logstash-${ELASTIC_VERSION}.tar.gz -C /usr/share && \
    mv /usr/share/logstash-${ELASTIC_VERSION} /usr/share/logstash && \
    chown --recursive logstash:logstash /usr/share/logstash/ && \
    ln -s /usr/share/logstash /opt/logstash

WORKDIR /usr/share/logstash

ENV ELASTIC_CONTAINER true
ENV PATH=/usr/share/logstash/bin:$PATH

# Provide a minimal configuration, so that simple invocations will provide
# a good experience.
ADD config/pipelines.yml config/pipelines.yml
ADD config/logstash-${IMAGE_FLAVOR}.yml config/logstash.yml
ADD config/log4j2.properties config/
ADD pipeline/default.conf pipeline/logstash.conf
ADD pki /etc/pki
RUN chown --recursive logstash:logstash config/ pipeline/

# Ensure Logstash gets a UTF-8 locale by default.
ENV LANG='en_US.UTF-8' LC_ALL='en_US.UTF-8'

HEALTHCHECK --interval=1s --retries=600 CMD curl -f http://localhost:9600/_node/stats

EXPOSE 5044 5055 9600

COPY docker-entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
CMD logstash
