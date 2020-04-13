#!/bin/sh

mkdir -p pki/tls/certs
mkdir -p pki/tls/private
openssl req -subj '/CN=sredis/' -x509 -days $((100 * 365)) -batch -nodes -newkey rsa:2048 -keyout pki/tls/private/sredis.key -out pki/tls/certs/sredis.crt
