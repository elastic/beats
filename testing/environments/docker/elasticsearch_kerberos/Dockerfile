FROM docker.elastic.co/elasticsearch/elasticsearch:7.8.0-SNAPSHOT

ADD scripts /scripts
ADD config /config
ADD healthcheck.sh /healthcheck.sh
ADD start.sh /start.sh

ENV REALM_NAME ELASTIC
ENV KDC_NAME elasticsearch_kerberos.elastic
ENV BUILD_ZONE elastic
ENV ELASTIC_ZONE $BUILD_ZONE

USER root
RUN /scripts/installkdc.sh && /scripts/addprincs.sh
USER elasticsearch
