#!/bin/sh

# check if service principal is OK
export KRB5_CONFIG=/etc/krb5.conf 
kinit -k -t /etc/HTTP_elasticsearch_kerberos.elastic.keytab HTTP/elasticsearch_kerberos.elastic@$REALM

# check if beats user can connect
kinit beats@$REALM
klist

curl --negotiate -u : -XGET http://elasticsearch_kerberos.elastic:9200/
