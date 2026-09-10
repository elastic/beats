# The current Elasticsearch image is RHEL-based, but its enabled UBI
# repositories do not provide the Kerberos KDC packages. Copy the current
# Elasticsearch distribution into Debian, where the fixture can install and
# run krb5kdc and kadmind alongside Elasticsearch.
FROM docker.elastic.co/elasticsearch/elasticsearch:9.6.0-7392fd3c-SNAPSHOT AS elasticsearch

FROM debian:bookworm

ENV PATH=/usr/share/elasticsearch/bin:$PATH
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl tini && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --non-unique --gid 0 elasticsearch && \
    useradd --uid 1000 --gid elasticsearch --home-dir /usr/share/elasticsearch --no-create-home elasticsearch

COPY --from=elasticsearch --chown=elasticsearch:root /usr/share/elasticsearch /usr/share/elasticsearch
COPY --from=elasticsearch /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

ADD scripts /scripts
ADD config /config
ADD healthcheck.sh /healthcheck.sh
ADD start.sh /start.sh

ENV REALM_NAME=elastic
ENV KDC_NAME=elasticsearch_kerberos.elastic
ENV BUILD_ZONE=elastic
ENV ELASTIC_ZONE=$BUILD_ZONE

USER root
RUN /scripts/installkdc.sh && /scripts/addprincs.sh
USER elasticsearch

WORKDIR /usr/share/elasticsearch
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/docker-entrypoint.sh"]
