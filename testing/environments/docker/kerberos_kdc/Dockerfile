FROM ubuntu:14.04
ADD scripts /scripts

ENV REALM_NAME ELASTIC
ENV KDC_NAME kerberos_kdc
ENV BUILD_ZONE elastic
ENV ELASTIC_ZONE $BUILD_ZONE

RUN echo kerberos_kdc.elastic > /etc/hostname && echo "127.0.0.1 kerberos_kdc.elastic" >> /etc/hosts
RUN bash /scripts/installkdc.sh

EXPOSE 88
EXPOSE 749

CMD sleep infinity
