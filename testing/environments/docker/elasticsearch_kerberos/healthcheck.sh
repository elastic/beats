#!/bin/sh

# check if service principal is OK
export KRB5_CONFIG=/etc/krb5.conf 
kinit -k -t /etc/HTTP_localhost.keytab HTTP/localhost@$REALM_NAME

# check if beats user can connect
printf '%s\n' testing | kinit beats@$REALM_NAME
klist

curl -f -u admin:testing http://localhost:9200/
