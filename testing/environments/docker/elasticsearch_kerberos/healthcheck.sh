#!/bin/sh

# check if service principal is OK
export KRB5_CONFIG=/etc/krb5.conf 
kinit -k -t /etc/HTTP_localhost.keytab HTTP/localhost@$REALM

# check if beats user can connect
kinit beats@$REALM
klist

curl --negotiate -u : -XGET http://localhost:9200/
