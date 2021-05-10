set -e

krb5kdc
kadmind

addprinc.sh HTTP/elasticsearch_kerberos.elastic
addprinc.sh beats testing
