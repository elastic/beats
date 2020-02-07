#!/bin/sh
# Take the certificates and create a DER format and create a sha256 of it and encode it to base 64
# https://www.openssl.org/docs/manmaster/man1/dgst.html
openssl x509 -in ca/ca.crt -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | openssl enc -base64
