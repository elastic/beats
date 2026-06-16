#!/bin/sh

# start Kerberos services
export KRB5_KDC_PROFILE="/var/kerberos/krb5kdc/kdc.conf"
krb5kdc
kadmind


# start ES
bin/elasticsearch-users useradd admin -r superuser -p testing | /usr/local/bin/docker-entrypoint.sh eswrapper
