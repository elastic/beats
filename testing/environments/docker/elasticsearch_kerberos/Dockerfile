FROM docker.elastic.co/elasticsearch/elasticsearch:7.15.2-SNAPSHOT

ADD scripts /scripts
ADD config /config
ADD healthcheck.sh /healthcheck.sh
ADD start.sh /start.sh

ENV REALM_NAME ELASTIC
ENV KDC_NAME elasticsearch_kerberos.elastic
ENV BUILD_ZONE elastic
ENV ELASTIC_ZONE $BUILD_ZONE

USER root
## Centos 7 EOL reached
## This fixes "Could not resolve host: mirrorlist.centos.org" error
RUN sed -i 's/mirrorlist/#mirrorlist/g' /etc/yum.repos.d/CentOS-* && \
    sed -i 's|#baseurl=http://mirror.centos.org|baseurl=http://vault.centos.org|g' /etc/yum.repos.d/CentOS-* && \
    /scripts/installkdc.sh && /scripts/addprincs.sh
USER elasticsearch
