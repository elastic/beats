FROM docker.elastic.co/elasticsearch/elasticsearch:8.0.0-SNAPSHOT

ADD scripts /scripts
ADD config /config
ADD healthcheck.sh /healthcheck.sh
ADD start.sh /start.sh

ENV REALM_NAME ELASTIC
ENV KDC_NAME elasticsearch_kerberos.elastic
ENV BUILD_ZONE elastic
ENV ELASTIC_ZONE $BUILD_ZONE

RUN bash /scripts/installkdc.sh
RUN bash /scripts/addprincs.sh
