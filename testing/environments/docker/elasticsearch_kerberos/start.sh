#!/bin/sh

# start Kerberos services
export KRB5_KDC_PROFILE="/var/kerberos/krb5kdc/kdc.conf"
krb5kdc
kadmind

echo elasticsearch_kerberos.elastic > /etc/hostname && echo "127.0.0.1 elasticsearch_kerberos.elastic" >> /etc/hosts

# start ES
bin/elasticsearch-users useradd admin -r superuser -p testing | /usr/local/bin/docker-entrypoint.sh eswrapper
