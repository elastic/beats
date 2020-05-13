#!/bin/sh

# start Kerberos services
krb5kdc
kadmind

# start ES
/usr/local/bin/docker-entrypoint.sh eswrapper
