#!/bin/sh

# start Kerberos services
export KRB5_KDC_PROFILE="/var/kerberos/krb5kdc/kdc.conf"
krb5kdc
kadmind


# start ES
bin/elasticsearch-users useradd admin -r superuser -p testing
exec /usr/local/bin/docker-entrypoint.sh eswrapper \
  -Etransport.host=127.0.0.1 \
  -Ehttp.host=0.0.0.0 \
  -Expack.license.self_generated.type=trial \
  -Expack.security.enabled=true \
  -Eindices.id_field_data.enabled=true \
  -Expack.security.audit.enabled=true \
  -Expack.security.authc.realms.kerberos.elastic.order=0 \
  -Expack.security.authc.realms.kerberos.elastic.keytab.path=/usr/share/elasticsearch/config/HTTP_localhost.keytab \
  -Expack.security.authc.realms.kerberos.elastic.remove_realm_name=false \
  -Expack.security.authc.realms.kerberos.elastic.krb.debug=true
