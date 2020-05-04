#!/bin/sh

# check if service principal is OK
KRB5_CONFIG=/etc/krb5.conf \
    kinit -k -t /etc/HTTP_elasticsearch_kerberos.elastic.keytab HTTP/elasticsearch_kerberos.elastic@ELASTIC


# check if beats user can connect
echo testing | KRB5_CONFIG=/etc/krb5.conf kinit beats@ELASTIC
klist
curl --negotiate -u : -XGET http://elasticsearch_kerberos.elastic:9200/
