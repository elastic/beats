FROM centos:8 AS builder
RUN yum update -qqy && yum install -qqy krb5-server krb5-libs krb5-workstation

FROM docker.elastic.co/elasticsearch/elasticsearch:8.0.0-SNAPSHOT

COPY --from=builder /usr/bin/kadmin  /usr/bin/kadmin
COPY --from=builder /usr/sbin/kadmin.local  /usr/sbin/kadmin.local
COPY --from=builder /usr/sbin/kadmind  /usr/sbin/kadmind
COPY --from=builder /usr/sbin/kdb5_util  /usr/sbin/kdb5_util
COPY --from=builder /usr/sbin/krb5kdc  /usr/sbin/krb5kdc

ADD scripts /scripts
ADD config /config
ADD healthcheck.sh /healthcheck.sh
ADD start.sh /start.sh

ENV REALM_NAME ELASTIC
ENV KDC_NAME elasticsearch_kerberos.elastic
ENV BUILD_ZONE elastic
ENV ELASTIC_ZONE $BUILD_ZONE

USER root
COPY --from=builder /usr/lib64 /tmp/usr/lib64
RUN cp -R -u /tmp/usr/lib64/* /usr/lib64 && rm -rf /tmp/usr

RUN /scripts/installkdc.sh && /scripts/addprincs.sh
USER elasticsearch
