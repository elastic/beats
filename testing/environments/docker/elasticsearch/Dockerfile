# Copy of https://github.com/elastic/elasticsearch-docker/blob/master/build/elasticsearch/Dockerfile
FROM centos:7
MAINTAINER Elastic Docker Team <docker@elastic.co>

ARG ELASTIC_VERSION
ARG DOWNLOAD_URL
ARG ES_JAVA_OPTS
ARG CACHE_BUST=1
ARG IMAGE_FLAVOR=x-pack

ENV ELASTIC_CONTAINER true
ENV PATH /usr/share/elasticsearch/bin:$PATH
ENV JAVA_HOME /usr/lib/jvm/java-1.8-openjdk

RUN yum update -y && yum install -y java-1.8.0-openjdk-headless wget which && yum clean all

RUN groupadd -g 1000 elasticsearch && adduser -u 1000 -g 1000 -d /usr/share/elasticsearch elasticsearch

WORKDIR /usr/share/elasticsearch

# Download/extract the defined ES version.
COPY download.sh /download.sh
RUN /download.sh $DOWNLOAD_URL $ELASTIC_VERSION $CACHE_BUST && rm /download.sh

RUN tar zxf elasticsearch-${ELASTIC_VERSION}.tar.gz && \
    chown -R elasticsearch:elasticsearch elasticsearch-${ELASTIC_VERSION} && \
    mv elasticsearch-${ELASTIC_VERSION}/* . && \
    rmdir elasticsearch-${ELASTIC_VERSION} && \
    rm elasticsearch-${ELASTIC_VERSION}.tar.gz

RUN set -e && for esdirs in config data logs; do \
        mkdir -p "$esdirs"; \
        chown -R elasticsearch:elasticsearch "$esdirs"; \
    done

USER elasticsearch

# Install plugins.
RUN elasticsearch-plugin install --batch ${DOWNLOAD_URL}/elasticsearch-plugins/ingest-user-agent/ingest-user-agent-${ELASTIC_VERSION}.zip
RUN elasticsearch-plugin install --batch ${DOWNLOAD_URL}/elasticsearch-plugins/ingest-geoip/ingest-geoip-${ELASTIC_VERSION}.zip

# Set bootstrap password (for when security is used)
RUN if [ "${IMAGE_FLAVOR}" = "x-pack" ]; then elasticsearch-keystore create; echo "changeme" | elasticsearch-keystore add -x 'bootstrap.password'; fi

COPY config/elasticsearch.yml config/
COPY config/log4j2.properties config/
COPY bin/es-docker bin/es-docker

USER root
RUN chown elasticsearch:elasticsearch config/elasticsearch.yml config/log4j2.properties bin/es-docker && \
    chmod 0750 bin/es-docker

# Enable a trial license for testing ML and Alerting.
RUN if [ "${IMAGE_FLAVOR}" = "x-pack" ]; then echo "xpack.license.self_generated.type: trial" >> config/elasticsearch.yml; fi

USER elasticsearch
CMD ["/bin/bash", "bin/es-docker"]

HEALTHCHECK --interval=1s --retries=600 CMD curl -f http://localhost:9200

EXPOSE 9200 9300
