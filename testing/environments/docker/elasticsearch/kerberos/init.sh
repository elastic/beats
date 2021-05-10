#!/bin/sh

# setup Keberos
echo elasticsearch_kerberos.elastic > /etc/hostname && echo "127.0.0.1 elasticsearch_kerberos.elastic" >> /etc/hosts

/scripts/installkdc.sh
/scripts/addprincs.sh

# add test user
bin/elasticsearch-users useradd beats -r superuser -p testing | /usr/local/bin/docker-entrypoint.sh eswrapper
