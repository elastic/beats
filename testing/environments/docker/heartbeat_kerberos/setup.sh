#!/bin/bash
# Provisions a throwaway MIT KDC, then starts the KDC and an SPNEGO-protected
# HTTP server (both in the foreground of this container).
#
# Client side (Heartbeat HTTP monitor) authenticates with password auth using a
# committed krb5.conf, so nothing needs to be copied out of the container.
#
# Principals:
#   testuser@EXAMPLE.COM        password: testpass   (client, password auth)
#   HTTP/localhost@EXAMPLE.COM  keytab at /etc/http.keytab (service)
set -euo pipefail

REALM="${REALM:-EXAMPLE.COM}"
CLIENT_USER="${CLIENT_USER:-testuser}"
CLIENT_PASS="${CLIENT_PASS:-testpass}"
HTTP_ADDR="${HTTP_ADDR:-:8080}"
ENCTYPES="aes256-cts-hmac-sha1-96:normal aes128-cts-hmac-sha1-96:normal"

mkdir -p /etc/krb5kdc /var/lib/krb5kdc

# Internal krb5.conf used by kadmin/krb5kdc inside the container (KDC on :88).
cat >/etc/krb5.conf <<EOF
[libdefaults]
  default_realm = ${REALM}
  dns_lookup_kdc = false
  dns_lookup_realm = false
  udp_preference_limit = 1
  default_tkt_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96
  default_tgs_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96
  permitted_enctypes = aes256-cts-hmac-sha1-96 aes128-cts-hmac-sha1-96

[realms]
  ${REALM} = {
    kdc = localhost:88
    admin_server = localhost:749
  }

[domain_realm]
  localhost = ${REALM}
  .localhost = ${REALM}
EOF

cat >/etc/krb5kdc/kdc.conf <<EOF
[kdcdefaults]
  kdc_ports = 88
  kdc_tcp_ports = 88

[realms]
  ${REALM} = {
    database_name = /var/lib/krb5kdc/principal
    admin_keytab = /etc/krb5kdc/kadm5.keytab
    acl_file = /etc/krb5kdc/kadm5.acl
    key_stash_file = /etc/krb5kdc/stash
    max_life = 10h 0m 0s
    max_renewable_life = 7d 0h 0m 0s
    supported_enctypes = ${ENCTYPES}
  }
EOF

echo "*/admin@${REALM} *" >/etc/krb5kdc/kadm5.acl

kdb5_util create -s -P masterpass

kadmin.local -q "addprinc -pw ${CLIENT_PASS} ${CLIENT_USER}@${REALM}"
kadmin.local -q "addprinc -randkey HTTP/localhost@${REALM}"
kadmin.local -q "ktadd -k /etc/http.keytab HTTP/localhost@${REALM}"

# Start the KDC in the background and wait for it to accept connections.
/usr/sbin/krb5kdc -n &
for _ in $(seq 1 30); do
  if bash -c "</dev/tcp/127.0.0.1/88" 2>/dev/null; then
    break
  fi
  sleep 0.5
done

echo "KDC_READY"
exec /usr/local/bin/spnego-server -addr "${HTTP_ADDR}" -keytab /etc/http.keytab
