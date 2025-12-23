set -e

export KRB5_KDC_PROFILE="/var/kerberos/krb5kdc/kdc.conf"
krb5kdc 
kadmind


## set principal and user
addprinc.sh HTTP/localhost
addprinc.sh beats testing
